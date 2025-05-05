

# Pebble Read Performance Based on Geth's Workflow

From [pebble.md](./pebble.md), we know that Pebble's performance varies based on different configurations. This document focuses on Pebble's read performance in the context of Geth's workflow.

## Geth's Workflow

We use `geth import` to simulate the block insertion process and analyze Pebble's performance during this operation.

**Note**: Currently, Geth doesn't have built-in database performance metrics. We've added metrics to Pebble and use [jsvisa/go-ethereum at state-reader-metric](https://github.com/jsvisa/go-ethereum/tree/state-reader-metric) to monitor performance.

The `geth import` command is configured as follows:

```bash
./bin/geth import \
    --db.engine=pebble \
    --datadir=data-base-xm \
    --cache=80960 \
    --cache.database=20 \
    --cache.snapshot=20 \
    --cache.gc=1 \
    --cache.trie=50 \
    --metrics --metrics.addr=0.0.0.0 --metrics.port=7540 \
    --state.scheme=path \
    --history.transactions=235000 \
    --history.state=90000 \
    --nocompaction \
    /hdd/geth-export-full/geth.blocks.0m-1m.gz \
    /hdd/geth-export-full/geth.blocks.1m-2m.gz \
    ...
    >> logs/import.log 2>&1
```

The host has 128GB of memory, with 80GB allocated as follows:

- Pebble DB cache: 16GB (20% of 80GB)
- Snapshot fastcache: 16GB (20% of 80GB)
- Trie pathdb fastcache: 40GB (50% of 80GB)

We use the [grafana/geth-pebble.json](./grafana/geth-pebble.json) dashboard to monitor performance metrics. Here are the key findings from the Geth import process:

### Blockchain Metrics

> Chain Insertion Time

![image-20250416132213697](assets/image-20250416132213697.png)

The mean chain insertion time is 78.8ms, with a maximum of 593ms.

> Phase Timing Breakdown

![image-20250416132405071](assets/image-20250416132405071.png)

| Phase           | Time (ms) | Percentage |
| --------------- | --------- | ---------- |
| storage read    | 22.0      | 34%        |
| execution       | 21.0      | 32%        |
| account read    | 12.3      | 19%        |
| storage update  | 2.90      | 4%         |
| account update  | 2.01      | 3%         |
| chain write     | 1.45      | 2%         |
| snapshot commit | 0.96      | 1%         |
| state commit    | 0.91      | 1%         |
| validation      | 0.86      | 1%         |
| account hash    | 0.74      | 1%         |

The analysis shows that 53% of the time (34% + 19%) is spent on storage and account reads, making these areas prime candidates for performance optimization.

For storage reads, we observe:

- Mean read time: 22ms
- Minimum time: 5.32ms
- Maximum time: 216ms

The significant variation in read times indicates potential optimization opportunities.

### State Metrics

The state reader in Geth uses a CachingDB that combines readers from multiple sources:

1. Snapshot flat reader
2. Trie hashdb/pathdb reader

The implementation can be found in [core/state/database.go](https://github.com/ethereum/go-ethereum/blob/2c52922ab4567a8cccd63ced2e88e892123072c4/core/state/database.go#L175-L209):

```go
func (db *CachingDB) Reader(stateRoot common.Hash) (Reader, error) {
    var readers []StateReader

    // Set up the state snapshot reader if available
    if db.snap != nil {
        snap := db.snap.Snapshot(stateRoot)
        if snap != nil {
            readers = append(readers, newFlatReader(snap))
        }
    } else {
        reader, err := db.triedb.StateReader(stateRoot)
        if err == nil {
            readers = append(readers, newFlatReader(reader))
        }
    }

    tr, err := newTrieReader(stateRoot, db.triedb, db.pointCache)
    if err != nil {
        return nil, err
    }
    readers = append(readers, tr)

    combined, err := newMultiStateReader(readers...)
    if err != nil {
        return nil, err
    }
    return newReader(newCachingCodeReader(db.disk, db.codeCache, db.codeSizeCache), combined), nil
}
```

We track read depth using metrics in [core/state/reader.go](https://github.com/ethereum/go-ethereum/blob/2c52922ab4567a8cccd63ced2e88e892123072c4/core/state/reader.go#L344-L360):

```go
func (r *multiStateReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
    var errs []error
    count := 0
    defer func() {
        counter := metrics.GetOrRegisterCounter(fmt.Sprintf("state/read/storage/%d", count), nil)
        counter.Inc(1)
    }()
    for _, reader := range r.readers {
        count++
        slot, err := reader.Storage(addr, slot)
        if err == nil {
            return slot, nil
        }
        errs = append(errs, err)
    }
    return common.Hash{}, errors.Join(errs...)
}
```

In our test case, all data was returned from `read 1`, indicating that all storage and account data came from the snapshot flat reader.

And the snapshot consists of three layers:

1. Diff layer (the most recent 128 block states)
2. Disk layer fast cache (LRU cache)
3. On-disk rawdb (Pebble DB)

Read distribution charts:

> Account Read Distribution

![image-20250416132509023](assets/image-20250416132509023.png)

> Storage Read Distribution

![image-20250416132535099](assets/image-20250416132535099.png)

**Only 3.14% of account reads and 8.79% of storage reads hit the on-disk rawdb, with most data being served from the diff layer and disk layer fast cache.**

### Ethdb (Pebble) Metrics

The [ethdb/pebble](https://github.com/ethereum/go-ethereum/tree/master/ethdb/pebble) implementation provides detailed metrics for monitoring Pebble's workflow and performance.

Read and write request distribution:

![image-20250416132705848](assets/image-20250416132705848.png)

| Metric name | Mean(ops) | Min(ops) | Max(ops) |
| ----------- | --------- | -------- | -------- |
| write       | 61.5      | 9.44     | 100      |
| read 200    | 3130      | 585      | 6220     |
| read 404    | 2800      | 392      | 5720     |

Note: `read 200` indicates the key was retrieved in the db, while `read 404` indicates the key was not found.

Key observations:

1. Read operations (both 200 and 404) are significantly more frequent than writes (approximately 3100 times)
2. The frequency of successful and failed reads are roughly equal to 1:1 



#### Ethdb latency breakdown

Here comes the latency of each operation:

> Write Latency

![image-20250416132752184](assets/image-20250416132752184.png)

> Read 200 Latency

![image-20250416132815680](assets/image-20250416132815680.png)

> Read 404 Latency

![image-20250416132832009](assets/image-20250416132832009.png)

| Metric name | Mean(μs) | Min(μs) | Max(μs) |
| ----------- | -------- | ------- | ------- |
| write       | 25.5     | 17.5    | 42.8    |
| read 200    | 113      | 24.8    | 2300    |
| read 404    | 41.5     | 9.43    | 90.2    |

Key findings:

1. Write latency is consistently low (25.5μs) due to `sync: false` write options
2. Read latency is significantly higher than the write latency
3. Read 200 latency shows high variability, indicating potential optimization opportunities
4. Read 404 latency remains stable

In the meanwhile, we found that the read 200 latency seems has some relationship with the pebble compaction count:

![image-20250417091354024](assets/image-20250417091354024.png)

When compaction occurs, a large number of SST files need to be read and written to the next level, which consumes a lot of disk I/O bandwidth, and results in insufficient I/O for reading.

In our subsequent optimization process, we can consider how to reduce compaction, eg:

1. How to decrease the number of SST files that are required for each compaction?
2. Whether it is possible to perform compaction more quickly and smoothly?
3. Whether it is feasible to limit the bandwidth for read and compaction operations, eg: separate them into different read and write queues.



#### Ethdb write bandwidth

Here is the write bandwidth metrics, which is  calculated by the bytes flushed - bytes compacted. This metric is roughly equivalent to the amount of data written to the raw database.

![image-20250417100248273](assets/image-20250417100248273.png)

Key findings:

- Mean write bandwidth: 11MB/s
- Peak write bandwidth: 49MB/s
- The data writing is uneven, on avarage, there will be a peak period of writing every 10minuts or so

## Pebble Read Benchmark

Our initial read benchmark used a stable database without writes, which doesn't accurately reflect Geth's workload. We need to simulate a more realistic scenario:

1. Initialize the database with a substantial size (e.g., 700GB) using key-value sizes similar to Geth's
2. Benchmark read performance with 50% existing keys and 50% non-existent keys
3. Run a sidecar write thread to simulate database mutations
4. Measure read performance under these conditions

### 1. Database Initialization

We're utilizing the  [cmd/db-iterator](https://github.com/jsvisa/go-ethereum/blob/db-iterator/cmd/db-iterator/analyze.py) tool to analyze a Geth's chaindb, the node is:

1. Ethereum full node
2. path scheme
3. db size is ~390GB

The results reveal the following key-value distribution:

> Key length distribution

| Length | Count      | Percentage |
| ------ | ---------- | ---------- |
| 33     | 3044938260 | 47.52      |
| 65     | 1255816825 | 19.60      |
| 38     | 405770871  | 6.33       |
| 37     | 398349989  | 6.22       |
| 39     | 353845432  | 5.52       |
| 36     | 210959913  | 3.29       |
| 8      | 173793595  | 2.71       |
| 9      | 172056223  | 2.69       |
| 40     | 158045169  | 2.47       |
| 35     | 83748492   | 1.31       |

> Value length distribution

| Length | Count      | Percentage(%) | Cumulative(%) |
| ------ | ---------- | ------------- | ------------- |
| <16B   | 3762937229 | 58.74         | 58.74         |
| <64B   | 1623984858 | 25.34         | 84.08         |
| <128B  | 855029504  | 13.33         | 97.41         |
| <1KB   | 164608814  | 2.51          | 99.92         |
| <4KB   | 351524     | 0.00          |               |
| <8KB   | 391346     | 0.00          |               |
| <1MB   | 466160     | 0.00          |               |
| >1MB   | 2          | 0.00          |               |

Keypoints:

1. Most keys are 33 or 65 bytes (snapshot keys, transaction lookups, block hashes, ...), 
2. Most values are less than 16 bytes.

With those results, then we initialize the database with 65-byte key length and 3-5 byte value length:

```bash
for b in {1..15}; do 
	pdb-writebench -keysize 65b -valuesize ${b}b -dir /md0/pb-dataset -test batch-100kb-mt-1gb-cache-04gb -size 20gb -keydir /md1/pb-keys -logdir pb-testlogs
done
```

After the write process, we go about ~300GB data in the database.

Now let's start the pure db read on that db, we start to test the read performance of different pebble configurations:

```bash
pdb-readbench -keysize 65b -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 50mb -keyrandom 50 -test geth-read-cache-02gb,pebble-read-cache-02gb,random-read-cache-02gb
```

Here we choose 3 read testcases:

1. `geth-read-cache-02gb`: with geth's pebble options, and the cache size is 2GB(same to the default value of the full node)
2. `pebble-read-cache-02gb`: use [pebble db's options](https://github.com/cockroachdb/pebble/blob/master/cmd/pebble/db.go#L57-L97), cache size the same as geth's
3. `random-read-cache-02gb`: use the default `pebble.Options{}`, cache size the same as geth's

> Read QPS

![image-20250418231632249](assets/image-20250418231632249.png)

> Read Latency

![image-20250418231835387](assets/image-20250418231835387.png)

Key findings:

1. Geth Read and Pebble's QPS and latency is similar
2. Geth's read is more stable, the derivation of read latency is small, while the Pebble's not
3. The mean read time is 200us, which is larger the ethdb's read performance in go-ethereum's 



Then test with `-sidewrite -valuesize 1mb` 

```bash
pdb-readbench -sidewrite -keysize 65b -valuesize 1mb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 100mb -keyrandom 50 -test geth-read-cache-02gb,pebble-read-cache-02gb,random-read-cache-02gb
```



>  The write bandwidth is about 5mb/s:

![image-20250419101918016](assets/image-20250419101918016.png)

>  Write QPS

![image-20250419102300283](assets/image-20250419102300283.png)

After the write process, the db reached to ~400GB

Then restart the same three read test case:

> Read QPS

![image-20250419065625102](assets/image-20250419065625102.png)

> Read Latency

![image-20250419065734909](assets/image-20250419065734909.png)

Key findings:

1. The read latency(~200-500us) for all the three test cases are all larger then the previous ones.(~150-200us)
2. The write is stable, which is not similar to geth's workflow, so need to take some adjusts



We introduce a burst write inside the side write, which will trigger a 500mb write every 5minutes(later changed this interval to every 1minute), then retest it with a large write valuesize:

```bash
pdb-readbench -sidewrite -keysize 65b -valuesize 2mb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-read-cache-02gb,pebble-read-cache-02gb,random-read-cache-02gb
```



The write QPS and latency as below:

![image-20250419104005250](assets/image-20250419104005250.png)

![image-20250419104016506](assets/image-20250419104016506.png)

Read QPS and latency as below:

![image-20250419104050863](assets/image-20250419104050863.png)

![image-20250419104124551](assets/image-20250419104124551.png)



Key findings:

1. The read latency is 500us-1ms, which is 2times more compared to the previous testcase.



In the mean while, let's test the db in read-only mode without the `-sidewrite` :

```bash
pdb-readbench -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-read-cache-02gb

# pebble manually compaction ~30mins

pdb-readbench -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-read-cache-02gb
```



![image-20250422055841915](assets/image-20250422055841915.png)

![image-20250422055919522](assets/image-20250422055919522.png)

![image-20250422060249612](assets/image-20250422060249612.png)

Key points:

1. Pre compaction, the read performance is not stable, but increases at the end of the first read phase, maybe the previous reads triggered the db compaction
2. Post compaction, the read latency is really low after the compaction, about 7 times lower compared the pre compaction



Apart from the 3-5 byte state db, we also write a bunch of data ranging from 0 to 128kb to simulate the block data.

```bash
pdb-writebench -keysize 65b -valuesize 10kb -dir /md0/pb-dataset -test batch-100kb-mt-1gb-cache-04gb -size 100gb -keydir /md1/pb-keys -logdir pb-testlogs
pdb-writebench -keysize 65b -valuesize 64kb -dir /md0/pb-dataset -test batch-100kb-mt-1gb-cache-04gb -size 100gb -keydir /md1/pb-keys -logdir pb-testlogs
pdb-writebench -keysize 65b -valuesize 128kb -dir /md0/pb-dataset -test batch-100kb-mt-1gb-cache-04gb -size 100gb -keydir /md1/pb-keys -logdir pb-testlogs
```

After this write process, we first manually compact the db, and then got about 900GB data in the database.

```bash
du -sh /md0/pb-dataset
900GB
```

So here we can start the read process to see how is it the read performance

```bash
pdb-readbench -keysize 65b -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 100mb -keyrandom 50 -test geth-read-cache-02gb,pebble-read-cache-02gb,random-read-cache-02gb
```



> Read QPS

![image-20250418101721923](assets/image-20250418101721923.png)

> Read Latency

![image-20250418101853211](assets/image-20250418101853211.png)

Key findings:

1. After compaction, the latency of DB read is relatively stable, and there is no significant relationship between the reading time and the size of the DB.



### Large dbsize test

In order to better test the scalability of Pebble, we wrote the data to the database up to 3TB(my ssd is 3.6TB), and then test it's read performance.  

We need to test based on geth's default pebble configuration, add the more test cases as below, ref https://github.com/jsvisa/goleveldb-bench/blob/5cccbcf5d23aca3a74713ce104691aee540b9292/cmd/pdb-readbench/pdb-readbench.go#L181-L387:

```go
- geth-default
- geth-MemTableSize-64mb
- geth-L0StopWritesThreshold-1000
- geth-L0CompactionThreshold-4
- geth-L0CompactionThreshold-12
- geth-level-BlockSize-32kb
- geth-level-BlockSize-32kb-IndexBlockSize-256kb
- geth-FlushSplitBytes-2mb
```



#### 3TB + sidewrite

First test with `-sidewrite` and without db compaction:

```bash
pdb-readbench -sidewrite -keysize 65b -valuesize 1kb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-FlushSplitBytes-2mb,geth-L0CompactionThreshold-4,geth-L0StopWritesThreshold-1000,geth-MemTableSize-64mb,geth-default,geth-level-BlockSize-32kb,geth-level-BlockSize-32kb-IndexBlockSize-256kb
```



> The read count and latency

![image-20250505210626537](assets/image-20250505210626537.png)

Key findings:

1. Read latency is really bad, mean time is 4.3ms, up to 6ms
2. The read performance is similar, no related to the different pebble configurations

#### 3TB + read-only

Then we test with the read-only mode, with a new `geth-optimized` [test case](https://github.com/jsvisa/goleveldb-bench/blob/5cccbcf5d23aca3a74713ce104691aee540b9292/cmd/pdb-readbench/pdb-readbench.go#L364-L388):

![image-20250505211051999](assets/image-20250505211051999.png)

Key findings:

1. Read latency is better than the sidewrite's case
2. The latency of read operations has been gradually reduced. This might be due to the block cache, as that compaction has not occurred



Later we manully compacting the 3TB, the log before and after as below:



```
2025/05/03 07:18:37 Before compaction metrics:
      |                             |       |       |   ingested   |     moved    |    written   |       |    amp
level | tables  size val-bl vtables | score |   in  | tables  size | tables  size | tables  size |  read |   r   w
------+-----------------------------+-------+-------+--------------+--------------+--------------+-------+---------
    0 |   100  228MB     0B       0 |  0.86 |    0B |     0     0B |     0     0B |     0     0B |    0B |   7  0.0
    1 |    18   64MB     0B       0 |  2.75 |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
    2 |   117  548MB     0B       0 |  1.05 |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
    3 |   604  4.6GB     0B       0 |  1.00 |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
    4 |  2.2K   40GB     0B       0 |  1.00 |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
    5 |  9.2K  338GB     0B       0 |  1.00 |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
    6 |   35K  2.8TB     0B       0 |     - |    0B |     0     0B |     0     0B |     0     0B |    0B |   1  0.0
total |   47K  3.2TB     0B       0 |     - |    0B |     0     0B |     0     0B |     0     0B |    0B |  13  0.0
-------------------------------------------------------------------------------------------------------------------
WAL: 1 files (0B)  in: 0B  written: 0B (0% overhead)
Flushes: 0
Compactions: 0  estimated debt: 8.2GB  in progress: 9 (0B)
             default: 0  delete: 0  elision: 0  move: 0  read: 0  rewrite: 0  multi-level: 0
MemTables: 1 (256KB)  zombie: 1 (4.0MB)
Zombie tables: 0 (0B)
Backing tables: 0 (0B)
Virtual tables: 0 (0B)
Block cache: 0 entries (0B)  hit rate: 0.0%
Table cache: 0 entries (0B)  hit rate: 0.0%
Secondary cache: 0 entries (0B)  hit rate: 0.0%
Snapshots: 0  earliest seq num: 0
Table iters: 0
Filter utility: 0.0%
Ingestions: 0  as flushable: 0 (0B in 0 tables)

2025/05/03 07:18:37 Compacting the database
2025/05/03 14:12:46 Compaction took 6h54m9.313546491s
2025/05/03 14:12:46 After compaction metrics:
      |                             |       |       |   ingested   |     moved    |    written   |       |    amp   |     multilevel
level | tables  size val-bl vtables | score |   in  | tables  size | tables  size | tables  size |  read |   r   w  |    top   in  read
------+-----------------------------+-------+-------+--------------+--------------+--------------+-------+----------+------------------
    0 |     0     0B     0B       0 |  0.00 |    0B |     0     0B |     0     0B |     0     0B |    0B |   0  0.0 |    0B    0B    0B
    1 |     0     0B     0B       0 |  0.00 | 228MB |     0     0B |     0     0B |    95  331MB | 331MB |   0  1.5 |    0B    0B    0B
    2 |     0     0B     0B       0 |  0.00 | 265MB |     0     0B |     0     0B |   186  925MB | 923MB |   0  3.5 |    0B    0B    0B
    3 |     0     0B     0B       0 |  0.00 | 689MB |     0     0B |     1   958B |   580  4.3GB | 4.3GB |   0  6.3 |  27MB 318MB 2.2GB
    4 |     0     0B     0B       0 |  0.00 | 4.0GB |     0     0B |     0     0B |  2.0K   30GB |  30GB |   0  7.3 | 153MB 1.6GB  13GB
    5 |     0     0B     0B       0 |  0.00 |  33GB |     0     0B |     2  2.1MB |  8.7K  250GB | 251GB |   0  7.5 | 1.4GB  16GB 132GB
    6 |   38K  3.1TB     0B       0 |     - | 382GB |     0     0B |     0     0B |   39K  3.1TB | 3.2TB |   1  8.4 |  12GB 136GB 1.1TB
total |   38K  3.1TB     0B       0 |     - |    0B |     0     0B |     3  2.1MB |   51K  3.4TB | 3.5TB |   1  0.0 |  13GB 154GB 1.2TB
---------------------------------------------------------------------------------------------------------------------------------------
WAL: 1 files (0B)  in: 0B  written: 0B (0% overhead)
Flushes: 0
Compactions: 10703  estimated debt: 0B  in progress: 0 (0B)
             default: 10700  delete: 0  elision: 0  move: 3  read: 0  rewrite: 0  multi-level: 1084
MemTables: 1 (256KB)  zombie: 1 (4.0MB)
Zombie tables: 0 (0B)
Backing tables: 0 (0B)
Virtual tables: 0 (0B)
Block cache: 0 entries (0B)  hit rate: 0.0%
Table cache: 0 entries (0B)  hit rate: 33.9%
Secondary cache: 0 entries (0B)  hit rate: 0.0%
Snapshots: 0  earliest seq num: 0
Table iters: 0
Filter utility: 0.0%
Ingestions: 0  as flushable: 0 (0B in 0 tables)
```



#### 3TB after compaction

Retest without `-sidewrite`:

```bash
pdb-readbench -keysize 65b -valuesize 1kb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-FlushSplitBytes-2mb,geth-L0CompactionThreshold-4,geth-L0StopWritesThreshold-1000,geth-MemTableSize-64mb,geth-default,geth-level-BlockSize-32kb,geth-level-BlockSize-32kb-IndexBlockSize-256kb
```

![image-20250505211750822](assets/image-20250505211750822.png)

Key findings:

1. The latency of the existing keys are really good, minimum to 50us
2. The latency of the not-found keys are 8 times worse than the existing keys, and the latency is stable, I think this maybe in the manully compaction, the bloom filter was destroyed?

Retest with `-sidewrite`, run with different `-valuesize 1kb, 100kb`:

```bash
pdb-readbench -sidewrite -keysize 65b -valuesize 1kb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-FlushSplitBytes-2mb,geth-L0CompactionThreshold-4,geth-L0StopWritesThreshold-1000,geth-MemTableSize-64mb,geth-default,geth-level-BlockSize-32kb,geth-level-BlockSize-32kb-IndexBlockSize-256kb

pdb-readbench -sidewrite -keysize 65b -valuesize 1kb -keydir /md1/pb-keys/batch-100kb-mt-1gb-cache-04gb -logdir pebble-read-logs -dir /md0/pb-dataset/testdb-batch-100kb-mt-1gb-cache-04gb/ -size 10mb -keyrandom 50 -test geth-FlushSplitBytes-2mb,geth-L0CompactionThreshold-4,geth-L0StopWritesThreshold-1000,geth-MemTableSize-64mb,geth-default,geth-level-BlockSize-32kb,geth-level-BlockSize-32kb-IndexBlockSize-256kb
```

![image-20250505212551499](assets/image-20250505212551499.png)

> 200 read latency
>
> ![image-20250505213011483](assets/image-20250505213011483.png)
>
> 404 read latency
>
> ![image-20250505213033765](assets/image-20250505213033765.png)

Key findings:

1. The read latency was worse than the readonly case

2. In the first `-sidewrite -valuesize 1kb` testcase, the read 200 latency is decreasing by time, it's wired

   

##### Conclusions

The root cause of read-related workload latency spikes is not the LSM-technology itself, but rather compaction related issues and the absence of advanced resource management (in RocksDB), and especially effective resource and QoS implementation and management.

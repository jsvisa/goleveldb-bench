# Shadow-Geth Grant Report: Database Analysis and Optimization for Go-Ethereum

## Executive Summary

This report presents the findings from the first two phases of the Shadow-Geth grant project, which focuses on evaluating and enhancing the scalability of Go-Ethereum (Geth)
with emphasis on handling significantly larger database sizes. The project aims to benchmark, analyze, and optimize the underlying database engine Pebble and Geth itself,
assessing their impact on overall performance under real-world workloads.

**Completed Phases:**

- **Phase 1**: Pebble Performance Benchmarking (4 weeks) ✅
- **Phase 2**: Chain Height and State Size Analysis (4 weeks) ✅
- **Phase 3**: Real-World Testing and Validation (8 weeks) ❌
- **Phase 4**: Database and Geth Optimizations (8 weeks) ❌

## Phase 1: Pebble Performance Benchmarking

### Objectives Achieved

✅ **Comprehensive Pebble Performance Testing**

- Successfully tested Pebble performance with datasets ranging from 100GB to 3TB
- Measured read/write performance under different workloads (read-only, read-write, mixed)
- Evaluated key metrics including throughput, latency, and disk I/O
- Documented correlation between dataset size and performance indicators
- Full report at [pebble-2.md](https://github.com/jsvisa/goleveldb-bench/blob/pebble/pebble-2.md)

### Key Infrastructure Developed

**Benchmark Tools Created:**

- `pdb-writebench`: Tool for benchmarking Pebble DB write performance
- `pdb-readbench`: Tool for benchmarking Pebble DB read performance in both read-only and read-write modes

**Monitoring Stack:**

- Prometheus for metrics collection
- Grafana for visualization with custom dashboards
- Node Exporter for system metrics
- Docker Compose setup for easy deployment

### Critical Performance Findings

#### 1. Geth Workflow Analysis

Using `geth import` to simulate real block insertion processes revealed:

- **Mean chain insertion time**: 78.8ms (max: 593ms)
- **Performance bottlenecks identified**:
  - Storage reads: 22.0ms (34% of total time)
  - Account reads: 12.3ms (19% of total time)
  - **Combined read operations account for 53% of block processing time**

#### 2. Database Operation Characteristics

**Read vs Write Frequency:**

- Read operations are ~3,100x more frequent than writes
- Read 200 (found): 3,130 ops/s average
- Read 404 (not found): 2,800 ops/s average
- Write operations: 61.5 ops/s average

**Latency Analysis:**

- Write latency: 25.5μs (consistently low due to async writes)
- Read 200 latency: 113μs (high variability: 24.8μs - 2.3ms)
- Read 404 latency: 41.5μs (more stable)

#### 3. Database Size Impact on Performance

**Read-Only Workload Performance:**

| Database Size | Mean Latency (μs) | Performance Degradation |
| ------------- | ----------------- | ----------------------- |
| 100GB         | 149               | Baseline                |
| 500GB         | 194-269           | 1.3-1.8x slower         |
| 3TB           | 392               | 2.6x slower             |

**Read-Write Workload Performance:**

| Database Size | Mean Latency (μs) | Performance Degradation |
| ------------- | ----------------- | ----------------------- |
| 100GB         | 335-347           | Baseline                |
| 500GB         | 383-551           | 1.1-1.6x slower         |
| 3TB           | 624-691           | 1.8-2.0x slower         |

#### 4. Critical Performance Issues Identified

**Compaction Impact:**

- Significant read/write performance degradation during background compaction
- Read latency spikes correlate with compaction activity
- Before compaction: 2,130μs average latency
- After compaction: 383μs average latency (5.6x improvement)

**Cache Efficiency:**

- Only 3.14% of account reads hit on-disk rawdb
- Only 8.79% of storage reads hit on-disk rawdb
- Most data served from diff layer and disk layer fast cache

### Technical Achievements

1. **Enhanced Geth with Performance Metrics**: Integrated custom metrics into Pebble via [jsvisa/go-ethereum state-reader-metric branch](https://github.com/jsvisa/go-ethereum/tree/state-reader-metric)
2. **Comprehensive Benchmark Suite**: Developed pebble benchmark testing tool that simulate Geth's exact workflow and configuration via [jsvisa/goleveldb-bench pebble branch](https://github.com/jsvisa/goleveldb-bench/tree/pebble)
3. **Real-time Monitoring**: Created Grafana dashboards for continuous performance monitoring

## Phase 2: Chain Height and State Size Analysis

### Objectives Achieved

✅ **Comprehensive State Growth Analysis**

- Analyzed trie node growth patterns from 2015-2025
- Measured flat state growth focusing on storage slots and account data
- Established correlation between chain height, time, and state size
- Created predictive models for future state size growth
- Full report at [geth-state-changeset.md](https://github.com/jsvisa/goleveldb-bench/blob/pebble/geth-state-changeset.md)

### Key Infrastructure Developed

**State Tracking Implementation:**

- Created a live tracer to record the state diff via [eth,core: add a state size live tracer #31914](https://github.com/ethereum/go-ethereum/pull/31914), thanks to @gary's [state float branch](https://github.com/rjl493456442/go-ethereum/tree/state-float)
- Added state size live tracer capability to Geth
- Implemented JSON output format for state change tracking

**Data Collection and Analysis:**

- Full sync from genesis with state change tracking
- Monthly aggregated state change dataset
- Visualization tools for state growth trends

### Critical State Growth Findings

#### 1. Historical State Growth Patterns (2015-2025)

**Monthly Growth Rates by Category:**

| Year | States (GiB/mo) | Trienodes (GiB/mo) | Codes (GiB/mo) |
| ---- | --------------- | ------------------ | -------------- |
| 2015 | 0               | 0                  | 0              |
| 2016 | 0.01            | 0.03               | 0.01           |
| 2017 | 0.13            | 0.46               | 0.14           |
| 2018 | 0.49            | 1.92               | 0.31           |
| 2019 | 0.40            | 1.47               | 0.32           |
| 2020 | 0.60            | 2.18               | 0.35           |
| 2021 | 0.76            | 2.86               | 0.33           |
| 2022 | 1.06            | 3.74               | 0.39           |
| 2023 | 0.90            | 3.16               | 0.46           |
| 2024 | 0.59            | 2.18               | 0.31           |
| 2025 | 0.87            | 3.27               | 0.38           |

#### 2. Key Growth Insights

**Trienodes as Primary Growth Driver:**

- Trienodes consistently represent the largest contributor to state growth
- Recent 18 months average: 2.7 GiB/month
- Peak growth: 4.74 GiB in April 2025

**State Growth Patterns:**

- Recent 18 months average: 0.7 GiB/month
- Range: 0.49-1.28 GiB/month
- Correlation with major network events (DeFi Summer, NFT boom, Layer 2 adoption)

**Code Growth:**

- Smallest but steadily increasing category
- Recent average: 0.34 GiB/month
- Range: 0.20-0.56 GiB/month

#### 3. Network Event Correlations

**Historical Growth Drivers Identified:**

- **2017**: ICO boom and CryptoKitties caused first major spikes
- **2018-2020**: Stablecoin growth and early DeFi activity
- **2020-2022**: DeFi Summer and NFT boom caused sustained high growth
- **2022-2025**: Continued DeFi, Layer 2s, and bridge adoption

#### 4. Future Growth Projections

**Conservative Estimates (based on 2024-2025 trends):**

- **Trienodes**: 2.7 GiB/month → 32.4 GiB/year
- **States**: 0.7 GiB/month → 8.4 GiB/year
- **Codes**: 0.34 GiB/month → 4.1 GiB/year
- **Total**: ~45 GiB/year combined growth

**Implications for Node Operators:**

- Storage requirements will continue growing substantially
- Performance optimization becomes increasingly critical
- Database efficiency improvements have compounding benefits

## Key Findings and Implications

### Performance Bottlenecks Identified

1. **Read Operations Dominate**: 53% of block processing time spent on database reads
2. **Compaction Impact**: Background compaction causes 5.6x performance degradation
3. **Scale Sensitivity**: Performance degrades significantly with database size (2.6x slower at 3TB)
4. **Non-existent Key Reads**: Substantial overhead from reads for non-existent items
5. **Application-Level Inefficiencies**: Identified multiple optimization opportunities within Geth code

### State Growth Concerns

1. **Accelerating Growth**: State size growing at ~45 GiB/year with increasing volatility
2. **Trie Node Dominance**: Trienodes account for majority of state growth
3. **Network Event Sensitivity**: Major ecosystem developments cause growth spikes

### Optimization Strategies Implemented

1. **Application-Level Focus**: Shifted from database tuning to Geth code optimization
2. **Targeted Performance Improvements**: Multiple pull requests addressing specific bottlenecks
3. **Code-Level Enhancements**: Direct improvements to Geth's performance-critical paths
4. **Practical Implementation**: Focus on achievable, measurable performance gains

## Phase 3: Real-World Testing and Validation

❌ **Real-World Testing and Validation**
✅ **Collaborative Development with Bloatnet Project**

As @CPerezz and other members are already working on the shadow state testing, so I collaborated with them to develop a custom Geth client for large-scale database testing,
the main contributes as below:

- [#542 · gballet/go-ethereum](https://github.com/gballet/go-ethereum/pull/542): support run geth with `--bloatnet` override flag
- [#545 · gballet/go-ethereum](https://github.com/gballet/go-ethereum/pull/545): add state size metrics, used to measure the state size growth
- [#546 · gballet/go-ethereum](https://github.com/gballet/go-ethereum/pull/546): fix the issue of large state size can panic geth

## Phase 4: Database and Geth Optimizations

❌ **Database and Geth Optimizations **

Because my skills do not lie in the optimization of the database's underlying structure, and we're going to use the Block Access List (BAL) to mitigate the read amplification issue,
so I shifted my focus from database-level optimizations to application-level performance improvements in Geth.

In this phase, I identified and addressed some performance bottlenecks within Geth code, especially in the pathdb indexing module,
and also find some bugs in the history state indexing module.

✅ **Pull Requests and Code Contributions**

Here are the full list of merged pull requests and code contributions made during the project:

1. [#31293](https://github.com/ethereum/go-ethereum/pull/31293) cmd/geth: update geth subcommand arguments #31293
2. [#31295](https://github.com/ethereum/go-ethereum/pull/31295) core/rawdb,state: add preimage miss metric #31295
3. [#31352](https://github.com/ethereum/go-ethereum/pull/31352) cmd/utils: set name to chaindata for all the opened db #31352
4. [#31353](https://github.com/ethereum/go-ethereum/pull/31353) ethdb: no need to check the metric nilness #31353
5. [#31360](https://github.com/ethereum/go-ethereum/pull/31360) cmd/geth: fix ctrl-c interrupt in import command #31360
6. [#31493](https://github.com/ethereum/go-ethereum/pull/31493) core/state: the metric of accountLoaded added once more #31493
7. [#31501](https://github.com/ethereum/go-ethereum/pull/31501) accounts/abi/abigen: fix a flaky bind test case NewSingleStructArgument #31501
8. [#31534](https://github.com/ethereum/go-ethereum/pull/31534) cmd: apply snapshot cache flag #31534
9. [#31577](https://github.com/ethereum/go-ethereum/pull/31577) cmd/geth: set trie,gc and other cache flags for import chain #31577
10. [#31646](https://github.com/ethereum/go-ethereum/pull/31646) cmd/geth: bloomfilter.size not used in the geth main process #31646
11. [#31800](https://github.com/ethereum/go-ethereum/pull/31800) core: use unix time to check fork readiness #31800
12. [#32012](https://github.com/ethereum/go-ethereum/pull/32012) trie: no need to store preimage if not enabled #32012
13. [#32060](https://github.com/ethereum/go-ethereum/pull/32060) triedb/pathdb: introduce file-based state journal #32060
14. [#32062](https://github.com/ethereum/go-ethereum/pull/32062) eth, core: terminate the downloader immediately when shutdown signal is received #32062
15. [#32067](https://github.com/ethereum/go-ethereum/pull/32067) eth: quick canceling block inserting when debug_setHead is invoked #32067
16. [#32080](https://github.com/ethereum/go-ethereum/pull/32080) eth/tracers: prestate lookup EIP7702 delegation account #32080
17. [#32093](https://github.com/ethereum/go-ethereum/pull/32093) all: replace override.prague with osaka #32093
18. [#32104](https://github.com/ethereum/go-ethereum/pull/32104) triedb: reset state indexer after snap synced #32104
19. [#32130](https://github.com/ethereum/go-ethereum/pull/32130) core/rawdb, triedb/pathdb: fix two inaccurate comments #32130
20. [#32219](https://github.com/ethereum/go-ethereum/pull/32219) triedb/pathdb: improve the performance of parse index block #32219
21. [#32226](https://github.com/ethereum/go-ethereum/pull/32226) triedb/pathdb: no need to reread metadata again #32226
22. [#32248](https://github.com/ethereum/go-ethereum/pull/32248) triedb/pathdb: fix incorrect address length in history searching #32248
23. [#32250](https://github.com/ethereum/go-ethereum/pull/32250) triedb/pathdb: use binary.append to eliminate the tmp scratch slice #32250
24. [#32260](https://github.com/ethereum/go-ethereum/pull/32260) triedb/pathdb: fix an deadlock when shorten a non fully indexed history #32260

And here are the still work in process pull requests:

1. [#32272](https://github.com/ethereum/go-ethereum/pull/) triedb/pathdb: add cache for history index and block #32272
2. [#32234](https://github.com/ethereum/go-ethereum/pull/) triedb/pathdb: rm the useless accountList and storageList #32234
3. [#32161](https://github.com/ethereum/go-ethereum/pull/) triedb: track and index the trienode in pathdb lookup #32161
4. [#32139](https://github.com/ethereum/go-ethereum/pull/) core,triedb: prevent debug_setHead from rewinding below oldest available state block #32139
5. [#32132](https://github.com/ethereum/go-ethereum/pull/) triedb,core: implement the partial state index read #32132
6. [#32101](https://github.com/ethereum/go-ethereum/pull/) eth: reset skeleton after chain rewinded #32101
7. [#32069](https://github.com/ethereum/go-ethereum/pull/) triedb: truncate history states in a batch way #32069
8. [#32362](https://github.com/ethereum/go-ethereum/pull/) core: state size metrics #32362

## In the end

### Acknowledgments

I would like to express my sincere gratitude to several individuals who made this project possible:

**@s1na** - Thank you for your invaluable assistance in applying for this grant. Your guidance through the application process was instrumental in getting this research off the ground.
**@gary** - Thank you for your continuous support and technical guidance throughout the entire project. Your expertise in system design and performance analysis provided crucial direction during challenging phases of the research. Your code review and comments significantly improved the quality of this work.

### Personal Reflections and Learning Experience

This grant project has been an incredibly enriching experience that deepened my understanding of Ethereum's state database architecture and performance characteristics.
Through this work, I gained substantial technical knowledge in several key areas, especially the **StateDB Architecture**.
Working extensively with Geth's state management system provided deep insights into how Ethereum handles account states, storage tries, and state transitions.
Understanding the intricate relationship between the state trie structure and database performance was particularly enlightening.

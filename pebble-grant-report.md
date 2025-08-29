# Shadow-Geth Grant Report: Database Analysis and Optimization for Go-Ethereum

## Executive Summary

This report presents the findings from the Shadow-Geth grant project, which focuses on evaluating and enhancing the scalability of Go-Ethereum (Geth) with emphasis on handling significantly larger database sizes. The project aimed to benchmark, analyze, and optimize both the underlying database engine (Pebble) and Geth itself, assessing their impact on overall performance under real-world workloads.

### Completed Phases

- **Phase 1**: Pebble Performance Benchmarking (4 weeks)
- **Phase 2**: Chain Height and State Size Analysis (4 weeks)
- **Phase 3**: Collaborative Development with Bloatnet Project (8 weeks)
- **Phase 4**: Geth Contributions (8 weeks)

## Phase 1: Pebble Performance Benchmarking

### Objectives Achieved

#### Comprehensive Pebble Performance Testing

- Successfully tested Pebble performance with datasets ranging from 100GB to 3TB
- Measured read/write performance under different workloads (read-only, read-write)
- Evaluated key metrics including throughput, latency, and disk I/O
- Documented correlation between dataset size and performance indicators
- Full report at [pebble-2.md](https://github.com/jsvisa/goleveldb-bench/blob/pebble/pebble-2.md)

### Key Infrastructure Developed

#### Benchmark Tools Created

- `pdb-writebench`: Tool for benchmarking Pebble DB write performance
- `pdb-readbench`: Tool for benchmarking Pebble DB read performance in both read-only and read-write modes

#### Monitoring Stack

- Prometheus for metrics collection
- Grafana for visualization with custom dashboards
- Node Exporter for system metrics
- Docker Compose setup for easy deployment

### Critical Performance Findings

#### 1. Geth Workflow Analysis

Using `geth import` (geth v1.15.7) to simulate real block insertion processes revealed:

- **Mean chain insertion time**: 78.8ms (max: 593ms)
- **Performance bottlenecks identified**:
  - Storage reads: 22.0ms (34% of total time)
  - Account reads: 12.3ms (19% of total time)
  - **Combined read operations account for 53% of block processing time**

#### 2. Database Operation Characteristics

#### Read vs Write Frequency

- Read operations are approximately 3,100× more frequent than writes
- Successful reads (found): 3,130 operations/second average
- Failed reads (not found): 2,800 operations/second average
- Write operations: 61.5 operations/second average

_Note: Write operations in Geth are performed in batch mode, resulting in significantly lower write operation counts compared to read operations._

#### Latency Analysis

- Write latency: 25.5μs (consistently low due to asynchronous writes)
- Successful read latency: 113μs (high variability: 24.8μs - 2.3ms)
- Failed read latency: 41.5μs (more stable performance)

#### 3. Database Size Impact on Performance

#### Read-Only Workload Performance

| Database Size | Mean Latency (μs) | Performance Degradation |
| ------------- | ----------------- | ----------------------- |
| 100GB         | 149               | Baseline                |
| 500GB         | 194-269           | 1.3-1.8x slower         |
| 3TB           | 392               | 2.6x slower             |

#### Read-Write Workload Performance

| Database Size | Mean Latency (μs) | Performance Degradation |
| ------------- | ----------------- | ----------------------- |
| 100GB         | 335-347           | Baseline                |
| 500GB         | 383-551           | 1.1-1.6x slower         |
| 3TB           | 624-691           | 1.8-2.0x slower         |

#### 4. Critical Performance Issues Identified

#### Compaction Impact

- Significant read/write performance degradation during background compaction
- Read latency spikes correlate with compaction activity
- Before compaction: 2,130μs average latency
- After compaction: 383μs average latency (5.6x improvement)

#### Cache Efficiency

- Only 3.14% of account reads hit on-disk rawdb
- Only 8.79% of storage reads hit on-disk rawdb
- Most data served from diff layer and disk layer fast cache

### Technical Achievements

#### Key Deliverables

1. **Enhanced Geth with Performance Metrics**: Integrated custom metrics into Pebble via [jsvisa/go-ethereum state-reader-metric branch](https://github.com/jsvisa/go-ethereum/tree/state-reader-metric)
2. **Comprehensive Benchmark Suite**: Developed pebble benchmark testing tool that simulate Geth's exact workflow and configuration via [jsvisa/goleveldb-bench pebble branch](https://github.com/jsvisa/goleveldb-bench/tree/pebble)
3. **Real-time Monitoring**: Created Grafana dashboards for continuous performance monitoring

## Phase 2: Chain Height and State Size Analysis

### Objectives Achieved

#### Comprehensive State Growth Analysis

- Analyzed trie node growth patterns from 2014.07 to 2025.06
- Measured flat state growth focusing on storage slots and account data
- Established correlation between chain height, time, and state size
- Developed predictive models for future state size growth
- Full report at [geth-state-changeset.md](https://github.com/jsvisa/goleveldb-bench/blob/pebble/geth-state-changeset.md)

### Key Infrastructure Developed

#### State Tracking Implementation

- Created a live tracer to record the state diff via [eth,core: add a state size live tracer #31914](https://github.com/ethereum/go-ethereum/pull/31914), thanks to @gary's [state float branch](https://github.com/rjl493456442/go-ethereum/tree/state-float)
- Added state size live tracer capability to Geth
- Implemented JSON output format for state change tracking

#### Data Collection and Analysis

- Full sync from genesis with state change tracking
- Monthly aggregated state change dataset
- Visualization tools for state growth trends

### Critical State Growth Findings

#### 1. Historical State Growth Patterns (2015-2025)

Here is the stacked size of Ethereum state size over time with stacked categories:

- Account Size: the flat Account snapshot size
- Storage Size: the flat Storage snapshot size
- Trienode Size: the total size of all trienodes, including account tries and storage tries
- Code Size: the total size of all contract codes

![assets/state-staked-over-time.png](https://github.com/jsvisa/goleveldb-bench/blob/pebble/assets/state-staked-over-time.png)

#### Yearly Growth Rates by Category

| Year | State Size | Trienode Size | Code Size |
| ---- | ---------- | ------------- | --------- |
| 2015 | 0.03       | 0.05          | 0.01      |
| 2016 | 0.19       | 0.40          | 0.16      |
| 2017 | 2.92       | 7.47          | 2.13      |
| 2018 | 10.0       | 22.63         | 3.60      |
| 2019 | 7.94       | 17.94         | 3.77      |
| 2020 | 11.51      | 26.17         | 4.30      |
| 2021 | 16.0       | 36.20         | 4.07      |
| 2022 | 22.05      | 45.77         | 4.78      |
| 2023 | 16.17      | 35.33         | 5.37      |
| 2024 | 10.55      | 26.08         | 3.63      |
| 2025 | 8.56       | 20.02         | 2.50      |

#### 2. Key Growth Insights

#### Trienodes as Primary Growth Driver

- Trienodes consistently represent the largest contributor to state growth, account for **63%** of total state growth (238.07 GB out of 378.25 GB total)
- Recent trend: 2.56 GiB/month average (26.08 GiB in 2024, 20.02 GiB projected for 2025)
- Trienode growth rate is **2.2x higher** than state data growth on average
- Peak trienode growth occurred in **2022** with 45.77 GB increase

#### State Growth Patterns

- Recent trend: 1.05 GiB/month average (10.55 GiB in 2024, 8.56 GiB projected for 2025)
- Correlation with major network events (DeFi Summer, NFT boom, Layer 2 adoption)

#### Code Growth

- Smallest but steadily increasing category
- Recent average: 0.34 GiB/month (based on 2024-2025 data)

#### 3. Network Event Correlations

#### Historical Growth Drivers Identified

- **2017**: ICO boom and CryptoKitties caused first major spikes
- **2018-2020**: Stablecoin growth and early DeFi activity
- **2020-2022**: DeFi Summer and NFT boom caused sustained high growth
- **2022-2025**: Continued DeFi, Layer 2s, and bridge adoption

#### 4. Future Growth Projections

#### Conservative Estimates (Based on Recent 2024-2025 Trends)

- **Trienodes**: 2.56 GiB/month → 30.72 GiB/year
- **States**: 1.05 GiB/month → 12.6 GiB/year
- **Codes**: 0.34 GiB/month → 4.1 GiB/year
- **Total**: ~48 GiB/year combined growth

#### Implications for Node Operators

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

1. **Accelerating Growth**: State size growing at ~48 GiB/year with increasing volatility
2. **Trie Node Dominance**: Trienodes account for majority of state growth
3. **Network Event Sensitivity**: Major ecosystem developments cause growth spikes

### Optimization Strategies Implemented

1. **Application-Level Focus**: Shifted from database tuning to Geth code optimization
2. **Targeted Performance Improvements**: Multiple pull requests addressing specific bottlenecks
3. **Code-Level Enhancements**: Direct improvements to Geth's performance-critical paths
4. **Practical Implementation**: Focus on achievable, measurable performance gains

## Phase 3: Collaborative Development with Bloatnet Project

The original plan was to conduct real-world testing and validation of performance improvements in Geth with large-scale datasets. Since @CPerezz and other team members were already working on shadow state testing, I collaborated with them to develop a custom Geth client for large-scale database testing.

### Key Contributions

The main contributions are as follows:

- [#542](https://github.com/gballet/go-ethereum/pull/542) - Support running Geth with `--bloatnet` override flag
- [#545](https://github.com/gballet/go-ethereum/pull/545) - Add state size metrics for measuring state growth
- [#546](https://github.com/gballet/go-ethereum/pull/546) - Fix panic issue when handling large state sizes

## Phase 4: Pull Requests and Code Contributions

Given my expertise in application-level optimization rather than database internals, and considering that the Block Access List (BAL) approach was planned to mitigate read amplification issues, I shifted focus from database-level optimizations to application-level performance improvements in Geth.

In this phase, I identified and addressed several performance bottlenecks within Geth code, particularly in the pathdb indexing module, and also discovered bugs in the history state indexing module.

Here is the complete list of merged pull requests and code contributions made during the project:

### Merged Pull Requests

#### Command Line and Configuration Improvements

1. [#31293](https://github.com/ethereum/go-ethereum/pull/31293) - Update geth subcommand arguments
2. [#31352](https://github.com/ethereum/go-ethereum/pull/31352) - Set name to chaindata for all opened databases
3. [#31360](https://github.com/ethereum/go-ethereum/pull/31360) - Fix Ctrl-C interrupt in import command
4. [#31534](https://github.com/ethereum/go-ethereum/pull/31534) - Apply snapshot cache flag
5. [#31577](https://github.com/ethereum/go-ethereum/pull/31577) - Set trie, GC and other cache flags for import chain
6. [#31646](https://github.com/ethereum/go-ethereum/pull/31646) - Fix bloomfilter.size not used in the geth main process

#### Metrics and Monitoring Enhancements

7. [#31295](https://github.com/ethereum/go-ethereum/pull/31295) - Add preimage miss metric
8. [#31353](https://github.com/ethereum/go-ethereum/pull/31353) - Remove unnecessary metric nilness check
9. [#31493](https://github.com/ethereum/go-ethereum/pull/31493) - Fix accountLoaded metric double counting

#### Core System Improvements

10. [#31501](https://github.com/ethereum/go-ethereum/pull/31501) - Fix flaky bind test case NewSingleStructArgument
11. [#31800](https://github.com/ethereum/go-ethereum/pull/31800) - Use unix time to check fork readiness
12. [#32012](https://github.com/ethereum/go-ethereum/pull/32012) - Skip storing preimage when not enabled
13. [#32062](https://github.com/ethereum/go-ethereum/pull/32062) - Terminate downloader immediately on shutdown signal
14. [#32067](https://github.com/ethereum/go-ethereum/pull/32067) - Quick canceling block insertion when debug_setHead is invoked
15. [#32080](https://github.com/ethereum/go-ethereum/pull/32080) - Prestate lookup EIP7702 delegation account
16. [#32093](https://github.com/ethereum/go-ethereum/pull/32093) - Replace override.prague with osaka
17. [#32104](https://github.com/ethereum/go-ethereum/pull/32104) - Reset state indexer after snap synced

#### PathDB Performance Optimizations

18. [#32060](https://github.com/ethereum/go-ethereum/pull/32060) - Introduce file-based state journal
19. [#32130](https://github.com/ethereum/go-ethereum/pull/32130) - Fix inaccurate comments in core/rawdb and triedb/pathdb
20. [#32219](https://github.com/ethereum/go-ethereum/pull/32219) - Improve performance of parse index block
21. [#32226](https://github.com/ethereum/go-ethereum/pull/32226) - Eliminate redundant metadata reads
22. [#32248](https://github.com/ethereum/go-ethereum/pull/32248) - Fix incorrect address length in history searching
23. [#32250](https://github.com/ethereum/go-ethereum/pull/32250) - Use binary.append to eliminate temporary scratch slice
24. [#32260](https://github.com/ethereum/go-ethereum/pull/32260) - Fix deadlock when shortening non-fully indexed history

The following pull requests are still work in progress:

### Work-in-Progress Pull Requests

1. [#32272](https://github.com/ethereum/go-ethereum/pull/32272) - Add cache for history index and block (triedb/pathdb)
2. [#32234](https://github.com/ethereum/go-ethereum/pull/32234) - Remove unused accountList and storageList (triedb/pathdb)
3. [#32161](https://github.com/ethereum/go-ethereum/pull/32161) - Track and index trienode in pathdb lookup (triedb)
4. [#32139](https://github.com/ethereum/go-ethereum/pull/32139) - Prevent debug_setHead from rewinding below oldest available state block (core,triedb)
5. [#32132](https://github.com/ethereum/go-ethereum/pull/32132) - Implement partial state index read (triedb,core)
6. [#32101](https://github.com/ethereum/go-ethereum/pull/32101) - Reset skeleton after chain rewind (eth)
7. [#32069](https://github.com/ethereum/go-ethereum/pull/32069) - Truncate history states in batch mode (triedb)
8. [#32362](https://github.com/ethereum/go-ethereum/pull/32362) - Add state size metrics (core)

## Conclusion

### Acknowledgments

I would like to express my sincere gratitude to the individuals who made this project possible:

- **@s1na** - Thank you for your invaluable assistance in applying for this grant. Your guidance through the application process was instrumental in getting this research off the ground.
- **@gary** - Thank you for your continuous support and technical guidance throughout the entire project. Your expertise in system design and performance analysis provided crucial direction during challenging phases of the research. Your code reviews and comments significantly improved the quality of this work.

### Personal Reflections and Learning Experience

This grant project has been an incredibly enriching experience that deepened my understanding of Ethereum's state database architecture and performance characteristics. Through this work, I gained substantial technical knowledge in several key areas, particularly in **StateDB Architecture**.

Working extensively with Geth's state management system provided deep insights into how Ethereum handles account states, storage tries, and state transitions. Understanding the intricate relationship between the state trie structure and database performance was particularly enlightening and will inform future optimization efforts in the Ethereum ecosystem.

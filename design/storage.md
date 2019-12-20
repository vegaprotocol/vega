# `BadgerDB`
## `Dir & ValueDir`
These specify the location of the `*.sst` and `*.vlog` files. By default we use the same folder but if we come up against IOP limits we can split them apart. For my testing they are set to the same location.

## `SyncWrites`
Does each write have to be flushed to disk.
This has a huge impact on the time it takes to handle small batches of inserts.

`Inserting 10K batches of 6 key value pairs:
* SyncWrites False: 0m01
* SyncWrites True (SHM): 0m01
* SyncWrites True (NVMe): 0m18
* SyncWrites True (SATA SSD): 0m55
* SyncWrites True (HD): 6m08`

## `TableLoadingMode & ValueLogLoadingMode`
The `BadgerDB` correctly uses virtual memory to map files into memory space and does not use up resident memory if the underlying data is not being accessed. Therefore it makes sense to use `MemoryMap` instead of `FileIO` to reduce system overhead.

## `NumVersionsToKeep`
We do not want to keep older versions of key value pairs so this should always be set to 1

## `ReadOnly`
False as we want to write to the DB

## `Truncate`
False as we never want to lose data on start-up

## `Logger`

## `Compression`
https://discuss.dgraph.io/t/badger-compression-feedback/5478/2

We should use snappy as it has the fastest compression/decompression speeds compared to the `zstd` version included in the package.

## `EventLogging`
Trace logging, useful for debugging.

## `MaxTableSize`
This is the size of the `sst` file stored on the disk, to reduce file handle usage we should use large files here.

## `LevelSizeMultiplier`
This defines how much each new level increases in size over the previous
e.g. a level 1 of size 10MB with a `LevelSizeMultiplier` of 10 would have a level 2 of size 100MB and a level 3 of size 1000MB

## `MaxLevels`
The maximum number of levels allowed. This value combined with the `LevelSizeMultiplier` gives the maximum size of the database. Trying to insert new pairs after the size limit is reached causes the application to crash. There is a hard limit to the size of the LSM.

## `ValueThreshold`
Values of this size or smaller are stored with the keys in the LSM object. Objects larger are stored in the `vlog` files.

## `NumMemtables`
The number of level 0 memory based tables

## `BlockSize`
Size of each block inside the `.sst`. This has no effect on the size of the files.
Compression is applied on a per block basis so the larger the block, the better the compression ratio.

## `BloomFalsePositive`
This changes how values are looked up. A lower number can result in more memory usage (but I don't know why)

## `KeepL0InMemory`
Force all of the L0 table to stay in memory. This is a new `v2` option as before the table could be paged out.

## `MaxCacheSize`
The size of the read cache used when performing lookups. I did not test this as I was only performing writes.

## `NumLevelZeroTables`
How many level 0 tables we have

## `NumLevelZeroTablesStall`
The number of level 0 tables we have to fill before inserts are blocked

## `LevelOneSize`
The size of our level 1 table. This value will be used with the `LevelSizeMultiplier` to define all the other level sizes

## `ValueLogFileSize`
The size of the `*.vlog` files. Making these large does not slow things down but will reduce the number of file handles used. The `BadgerDB` library has a maximum size of 2GB for no real reason.

## `ValueLogMaxEntries`
The maximum number of pairs we can store in a single `vlog` file. If we are trying to keep the file size constant, we should set this value to something huge so it is never used as a blocker to more writes occurring.

## `NumCompactors`
Compactors are used when pairs need to be moved down a level due to a level being full up. The more levels we have in the system, the more times `compactions` will occur before the values reach their last level. The more compactors we allow, the higher the possible memory usage will be if they are all used at the same time.

## `CompactL0OnClose`
Compact all the tables before the database is closed. We won't be closing the DB as it will be up 24/7 so this doesn't really matter.

## `LogRotatesToFlush`
For values that are large in size, this is useful to force the `memtables` to be flushed after a certain amount of `vlog` file rotates. As we have similar sized keys to values this doesn't matter so we can leave this as default.

## `VerifyValueChecksum`
Validate every read from the log file, but at the cost of CPU overhead. We don't need this

## `ZSTDCompressionLevel`
The default compression level is 15 which is very CPU hungry. If we decided to use this we should move towards the 1-5 end of the values to prevent it slowing us down.

# Changelog

## [0.2.0](https://github.com/duizendstra/alexandria/compare/go/platform/coordination/v0.1.0...go/platform/coordination/v0.2.0) (2026-08-22)


### ⚠ BREAKING CHANGES

* **coordination:** Excluder.Acquire(string) is now TryAcquire(Subject); NopExcluder is removed (nothing consumed it); filelock.Locker{Dir, Prefix, Signals, OnSignal} is replaced by filelock.Store{Dir, Purpose, Options}, and the signal re-raise is gone. go/platform/runstate keeps pinning coordination v0.1.0 until its own release adopts the new contract.

### Features

* **coordination:** adopt the waiter/excluder language with holder records, fences and opt-in reclaim ([#259](https://github.com/duizendstra/alexandria/issues/259)) ([4e2149d](https://github.com/duizendstra/alexandria/commit/4e2149d51485bde0fa5d4ac2c7a882b0e0305d2e))

## 0.1.0 (2026-08-22)

### Features

* **platform/coordination:** add Excluder interface, NopExcluder, and filelock implementation

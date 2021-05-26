# xk6-segment
Extracting and wrapping SegmentIndex from k6

</div>

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

1. Build with `xk6`:

```bash
xk6 build --with github.com/mstoykov/xk6-segment
```

This will result in a `k6` binary in the current directory.

2. Run with the just build `k6:

```bash
./k6 run test.js
```

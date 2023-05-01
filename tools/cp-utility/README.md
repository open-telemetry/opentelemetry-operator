# Introduction

This copy utility is intended to be used as a base image for OpenTelemetry Operator 
autoinstrumentation images. The copy utility will allow autoinstrumentation packages to be 
copied from the init container to the final destination volume.

## Development

### Pre-requirements
* Install rust

```
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

* Install rustfmt

```
rustup component add rustfmt
```

### Development

* Auto formatting the code

This step is important and it might fail the build if the files are not properly
formatted.

```
cargo fmt
```

* Testing the code
```
cargo test
```

* Building the code

```
cargo build
```

NOTE: this will build the code for tests locally. It will not statically link the libc used by it.


* Building the code statically linked

```
cargo build --target x86_64-unknown-linux-musl
```

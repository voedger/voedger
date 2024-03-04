#--wasm-abi=generic is needed to provide support of uint64 in WASM functions parameters
# By default parameters must be int32 because it supposed to work with javascript.
tinygo build --no-debug -o extnogc.wasm -scheduler=none -opt=2 -gc=leaking -target=wasi .


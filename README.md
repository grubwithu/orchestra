# Orchestra: Coordinating Multiple Fuzzers to Solve Code Constraint Bottlenecks
This is a module that runs with [pfuzzer](https://github.com/grubwithu/pfuzzer). It can tell pfuzzer which corpus to focus on.

## Usage

> Docker is recommended to run HFC.

Pre-requisite:
- Clang-21, LLVM-21, CMake, Build-essential, and so on...
- `llvm-profdata`, `llvm-opt`, and `llvm-cov` are needed in PATH.
- There is a environment variable `FUZZ_INTRO` to specify the path of `FuzzIntrospector.so`.

Init submodule:
```
$ git submodule update --init --recursive
```

Run the demo:

```
$ bash scripts/run_demo.sh freetype2
```

This script will:
- Build HFC in `build` directory.
- Compile pfuzzer in `pfuzzer/build` directory, and we can get `libfuzzer.a`.
- Build the target(freetype2) binary.
  - Generate static profiles of the target binary(using opt and fuzz-introspector). (We can get `test/freetype2/build__HFC_qzmp__/ftfuzzer` as A)
  - Compile the target binary for coverage instrumentation. (We can get `test/freetype2/build-runtime/ftfuzzer_cov` as B)
  - Link `libfuzzer.a` to the target binary. (We can get `test/freetype2/build-runtime/ftfuzzer` as C)
  - (In fact, three binaries are generated. A is never used. B is used for coverage report in HFC. C is running as pfuzzer.)
- Run HFC and pfuzzer in parallel.

## Run in Docker

We provide Dockerfiles to build the test environment.

```
$ docker build -t hfc-base:latest .
$ cd test/
$ docker build -t hfc-test:latest .
```

## TODO 26.1.11

1. [pfuzzer] Use command argument to specify fuzzing strategy. (DONE)
2. [pfuzzer] Add more fuzzing strategies.
3. [HFC] C++ support: parse name like `_ZN....`. (Go support is not complete.)
4. [HFC] Record coverage increase or decrease for every fuzzers. (DONE)

## TODO 26.1.19

1. [pfuzzer] Modify `GetJobSeeds` to support fuzzing strategy.
2. [HFC] Add more methods to choose constraint group 

## TODO 26.3.23

1. JSON Region (Un)covered count (DONE)
2. Module 1 Global JSON Calculation (DONE) -> Important Function CallTrace (DONE) -> Weight Calculation (DONE) -> Analysis Constraint Feature Count(AST) (DONE) -> Extract TOKEN 2 Dictionary 
3. Module 2 Incremental LineCov -> Fuzzer Constraint Weight Matrix(Constraint Weight, New Branch / JobBudget) (DONE)
4. Add JobID JobBudget (DONE)
5. CallTrace 2 Pfuzzer for seed distance calculation
6. FuzzerFork.cpp

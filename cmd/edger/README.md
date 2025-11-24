# Intro
Edger is a part of Heeus, is the controller, executed on the *EdgeNode*.
Edger is used to get information about the EdgeNode and to manage its state.
Heeus engineers accesses to the EdgeNode by Edger:
- to deploy docker stacks,
- to install/upgrade Edger binaries,
- to execute the shell commands.

# Functional Design
Function design is provided by Quick start + Command Line Reference.

# Quickstart
(TODO: as example see MDA `ctools` https://github.com/dmitrymolchanovky/ctool/tree/26200-check-arguments-and-make-cluster-json)

# Command Line Reference
```
$ edger --help
```

# Limitations

- If MicroController achieves the state and state can be "broken" somehow there should be a metric which reports about it
- MicroControllers should be as idempotent as possible

# Design

- [tdesign.md](tdesign.md)



# gridbach-core
Gridbach's core algorithmic logic

## Overview
`gridbach-core` is the core computational logic of Gridbach, a grid computing system that verifies the Goldbach conjecture updating the world record for the computational verification of the famous unsolved mathematical problem.

Please visit the following site for more information about Gridbach:
https://gridbach.com

The calculation processing in the production environment is integrated into the Gridbach system in the form of WASM (WebAssembly) developed in Go, but since it is difficult to release it to the general public in its original form, it is recreated as a command-line tool and released.

## Technology Stack
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)

## Demo
<img src="https://gridbach.com/img/gif/gridbach-core-demo.gif" alt="Demo" style="width: 70%;">

## Requirement
go version go1.23.4

## Usage
```bash
git clone git@github.com:nakatahr/gridbach-core.git
cd gridbach-core
go run .
```

## Calculation Logic
The calculation process consists of four main steps:

1. Generate prime numbers for use in step 3. using the Sieve of Eratosthenes - `CreateRoot()`
2. Generate reversed prime number sequences for use in step 4. - `CreateReverse()`
3. Execute the Sieve of Eratosthenes on a 100 million length interval specified by jobId - the first half of `SieveAndVerify()`
4. Verify the Goldbach conjecture through bitwise operations of prime number sequences and reversed prime number sequences - the latter half of `SieveAndVerify()`

## TODO
- Write a blog post to give detailed explanation about idea and technichs behind the algorithm
- Add comments in the codes for clarity

## Licence

[MIT](https://github.com/tcnksm/tool/blob/master/LICENCE)

## Author

Hiroaki Jay Nakata https://gridbach.com/

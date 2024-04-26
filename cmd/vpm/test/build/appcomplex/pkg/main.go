package main

import (
	"fmt"
	pkg1 "pkg1/pkg"
	pkg2 "pkg2/pkg"
)

func main() {
	pkg1.Pkg1Func()
	pkg2.Pkg2Func()
	fmt.Println("This file is included unless excludeFile is defined.")
}

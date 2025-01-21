#!/bin/bash

# this script is used to generate test object files and run simple-linker

test_name=$(basename $0 .sh)
test_path=out/tests/$test_name

mkdir -p $test_path

# using <<EOF it reads the texts starting from next line until EOF
# -c => without linking; -x(c) => specifies the input file language, which is C
# - => read from the standard input
# -o => provide custom names
cat <<EOF | $CC -xc - -c -o $test_path/a.o
#include <stdio.h>
int main(void) {
    printf("Hello World\n");
    return 0;
}
EOF

# all c object files are put under tests dir, and we run simple linker
# with object files as arguments
#./simple-linker $test_path/a.o

# use gcc linker to make a real executable
# -B. => find a linker in the current directory
# -B finds a program called ld, therefore we should make our executable name ld
$CC -B. -static $test_path/a.o -o $test_path/out


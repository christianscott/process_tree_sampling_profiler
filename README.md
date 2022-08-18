# pstree_prof

Start a process & sample the tree of child processes it spawns

## installation

```shell
$ go get github.com/christianscott/pstree_prof
```

## example

```sh
$ cat eg/test.sh
#! /usr/bin/env bash

node -e 'setTimeout(() => {}, 1000)'
python3 -c 'import time; time.sleep(2)'
sleep 3                                                                                                                                                                                                                                                                                                                                                         
$ ./pstree_prof -cmd 'bash eg/test.sh' -freq 100 -fmt count
pstree_prof: 2022/04/08 11:48:00 sampling every 10ms
pstree_prof: 2022/04/08 11:48:00 start of output from command:
pstree_prof: 2022/04/08 11:48:07 end of output from command
count   command
42      bash eg/test.sh
19      sleep 3
13      /usr/local/Cellar/python@3.9/3.9.10/Frameworks/Python.framework/Versions/3.9/Resources/Python.app/Contents/MacOS/Python -c import time; time.sleep(2)
8       node -e setTimeout(() => {}, 1000)
1       (Python)
```

## todo

- [x] add `-command` flag
- [ ] export traces using otel
- [ ] export traces to a flamegraph-compatible format (stack samples?)
- [ ] use `libproc.h` instead of `ps`

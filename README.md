# pstree_prof

Start a process & sample the tree of child processes it spawns

## example

```sh
$ ./pstree_prof -command 'bash eg/test.sh' -samplingInterval 10
pstree_prof: sampling every 10 ms
count   command
117     bash eg/test.sh
56      sleep 3
38      /usr/local/Cellar/python@3.9/3.9.10/Frameworks/Python.framework/Versions/3.9/Resources/Python.app/Contents/MacOS/Python -c import time; time.sleep(2)
20      node -e setTimeout(() => {}, 1000)
1       (Python)
```

## todo

- [x] add `-command` flag
- [ ] export traces using otel
- [ ] curses output to stderr while sampling

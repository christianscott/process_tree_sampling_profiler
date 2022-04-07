# pstree_prof

Start a process & sample the tree of child processes it spawns

## example

```sh
pstree_prof: 2022/04/08 09:48:31 sampling every 10ms
pstree_prof: 2022/04/08 09:48:31 start of output from command:
pstree_prof: 2022/04/08 09:48:37 end of output from command
count   command
126     bash eg/test.sh
60      sleep 3
42      /usr/local/Cellar/python@3.9/3.9.10/Frameworks/Python.framework/Versions/3.9/Resources/Python.app/Contents/MacOS/Python -c import time; time.sleep(2)
21      node -e setTimeout(() => {}, 1000)
1       (Python)
1       (node)
```

## todo

- [x] add `-command` flag
- [ ] export traces using otel

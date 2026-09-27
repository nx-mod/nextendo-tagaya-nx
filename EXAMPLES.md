# nextendo-tagaya-nx — example usage

`./run.sh` starts Tagaya (:8471). The console asks it for the latest version of
each title; without an answer it can nag "software update required".

```sh
./run.sh &
./test-versions.sh      # prints the list + its ETag, then shows the 304 path
```

Edit `versions.json` to change the list (an embedded default is used if it is
absent, so it always answers).

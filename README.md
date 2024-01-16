## Patternmatcher

This is a fork of Moby's [patternmatcher](https://github.com/moby/patternmatcher)
package that exposes private structures.

### go.mod

```sh
go mod edit -replace github.com/moby/patternmatcher@v0.6.0=github.com/goller/patternmatcher@main
go mod tidy
```

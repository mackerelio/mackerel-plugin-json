# How to release

[goxz](https://github.com/Songmu/goxz) and [ghr](https://github.com/tcnksm/ghr) are used to release.

## Release by TravisCI

1. Edit CHANGELOG.md, git commit, git push
2. `git tag vx.y.z`
3. git push --tags
4. Wait to build at https://travis-ci.org/mackerelio/mackerel-plugin-json
5. See https://github.com/mackerelio/mackerel-plugin-json/releases

Don't forget setting GITHUB_TOKEN as environment variables in TravisCI.  If you don't know how, see https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings .

## Release by manually

1. Install goxz and ghr by `make setup`
2. Edit CHANGELOG.md, git commit, git push
3. `git tag vx.y.z`
4. GITHUB_TOKEN=... script/release
5. See https://github.com/mackerelio/mackerel-plugin-json/releases

module github.com/bakanura/gjallarOS

go 1.26.0

require github.com/JadeOpenServices/oddc v0.0.0

// ODDC is vendored as a git subtree so builds and installs stay offline.
replace github.com/JadeOpenServices/oddc => ./oddc

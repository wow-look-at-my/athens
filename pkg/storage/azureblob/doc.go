/*
Package azureblob provides a storage driver to upload module files to an
Azure Blob Storage container.

The implementation is compiled only under the "azureblob" build tag (like the
other optional cloud storage backends). This file carries no build tag so the
package still resolves under a plain "go build ./..." / "go test ./..." -- the
go-toolchain coverage run -- without failing on "build constraints exclude all
Go files", while the actual driver builds only when the tag is set.
*/
package azureblob

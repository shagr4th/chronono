#/bin/sh
if [[ "$OSTYPE" == "darwin"* ]]; then

    export CGO_CFLAGS=-mmacosx-version-min=10.11
    export CGO_CPPFLAGS=-mmacosx-version-min=10.11
    export CGO_CXXFLAGS=-mmacosx-version-min=10.11
    export CGO_LDFLAGS=-mmacosx-version-min=10.11

    # https://dev.to/thewraven/universal-macos-binaries-with-go-1-16-3mm3
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -a -ldflags="-s -w -extldflags=-mmacosx-version-min=10.11" -o chronono_amd64
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -a -ldflags="-s -w -extldflags=-mmacosx-version-min=10.11" -o chronono_arm64
    lipo -create -output chronono chronono_amd64 chronono_arm64
    rm chronono_amd64 chronono_arm64

    rm -rf chronono.app
    mkdir chronono.app
    mkdir chronono.app/Contents
    mkdir chronono.app/Contents/MacOS
    mkdir chronono.app/Contents/Resources
    cp osx/Infoplist chronono.app/Contents/Info.plist
    mv chronono chronono.app/Contents/MacOS/
    cp osx/icon.icns chronono.app/Contents/Resources/icon.icns
else
    go build -a -ldflags="-s -w"
fi
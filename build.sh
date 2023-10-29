#/bin/sh
if [[ "$OSTYPE" == "darwin"* ]]; then

    export CGO_CFLAGS=-mmacosx-version-min=10.11
    export CGO_CPPFLAGS=$CGO_CPPFLAGS
    export CGO_CXXFLAGS=$CGO_CXXFLAGS
    export CGO_LDFLAGS=$CGO_LDFLAGS

    rm -rf chronono.app
    mkdir -p chronono.app/Contents/MacOS
    mkdir -p chronono.app/Contents/Resources
    cp osx/Infoplist chronono.app/Contents/Info.plist
    cp osx/icon.icns chronono.app/Contents/Resources/icon.icns
    
    # https://dev.to/thewraven/universal-macos-binaries-with-go-1-16-3mm3
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -a -ldflags="-s -w -extldflags=$CGO_LDFLAGS" -o chronono_amd64
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -a -ldflags="-s -w -extldflags=$CGO_LDFLAGS" -o chronono_arm64
    lipo -create -output chronono.app/Contents/MacOS/chronono chronono_amd64 chronono_arm64
    rm chronono_amd64 chronono_arm64
else
    go build -a -ldflags="-s -w"
fi
#/bin/sh

$GOPATH/bin/packr && go build -ldflags="-s -w"

if [[ "$OSTYPE" == "darwin"* ]]; then
    #brew install upx
    #disabled for the time being : https://github.com/upx/upx/issues/222
    #upx chronono
    rm -rf chronono.app
    mkdir chronono.app
    mkdir chronono.app/Contents
    mkdir chronono.app/Contents/MacOS
    mkdir chronono.app/Contents/Resources
    cp osx/Infoplist chronono.app/Contents/Info.plist
    cp chronono chronono.app/Contents/MacOS/
    cp osx/icon.icns chronono.app/Contents/Resources/icon.icns
fi
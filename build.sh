#brew install upx

$GOPATH/bin/packr && go build -ldflags="-s -w" && upx chronono
rm -rf chronono.app
mkdir chronono.app
mkdir chronono.app/Contents
mkdir chronono.app/Contents/MacOS
mkdir chronono.app/Contents/Resources
cp osx/Infoplist chronono.app/Contents/Info.plist
cp chronono chronono.app/Contents/MacOS/
cp osx/icon.icns chronono.app/Contents/Resources/icon.icns
NAME="SecretHub CLI"
EXE=bin/secrethub.exe
VERSION=$(git describe --tags $(git rev-list --tags --max-count=1))
MANUFACTURER=SecretHub
UPGRADE_CODE=7178CB18-B431-4A9D-B29E-D31E63A1CF77
ICON="/dist/secrethub.png"

curl -s -o dist/secrethub.png https://secrethub.io/img/secrethub-logo-rgb-shield-square.png
mkdir -p dist/msi
docker run -v $(pwd)/dist:/dist msi-packager /dist/windows_amd64 /dist/msi/secrethub_windows_amd64.msi -n $NAME -e $EXE -v $VERSION -m $MANUFACTURER -u $UPGRADE_CODE -i $ICON --local
docker run -v $(pwd)/dist:/dist msi-packager /dist/windows_386 /dist/msi/secrethub_windows_386.msi -n $NAME -e $EXE -v $VERSION -m $MANUFACTURER -u $UPGRADE_CODE -i $ICON --local

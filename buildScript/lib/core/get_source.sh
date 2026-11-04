#!/bin/bash
set -e

source "buildScript/init/env.sh"
ENV_NB4A=1
source "buildScript/lib/core/get_source_env.sh"
PATCH_FILE="$(pwd)/buildScript/patches/silikbox-core.patch"
pushd ..

####

if [ ! -d "sing-box-1.14.4" ]; then
  git clone --no-checkout https://github.com/SagerNet/sing-box.git sing-box-1.14.4
fi
pushd sing-box-1.14.4
git checkout "$COMMIT_SING_BOX"
if git apply --reverse --check "$PATCH_FILE" 2>/dev/null; then
  echo "NekoBox compatibility patch already applied"
else
  git apply --check "$PATCH_FILE"
  git apply "$PATCH_FILE"
fi
popd

popd

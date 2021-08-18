#!/usr/bin/env bash

CURR_DIR="$PWD"
CONDA_BIN="$HOME/miniconda/bin/conda"

if [[ -e "$HOME/miniconda" ]]; then
  echo "miniconda already installed"
  exit 0
fi

wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -O miniconda.sh
bash "${CURR_DIR}"/miniconda.sh -b

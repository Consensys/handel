#!/usr/bin/env bash


## REQUIREMENTS:
## python3, conda
##
## This script setups the python environment for the plots using conda
## It setups a "handel" envorionement, switch to it and install the required
## libraries

conda create --name handel
conda activate handel
## install the local conda version of pip
conda install --yes pip
pip install -r requirements.txt

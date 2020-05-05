#!/bin/bash
docker build . -f Dockerfile -t ryanlaycock/software-network-analysis-gateway:0.0.1
docker push ryanlaycock/software-network-analysis-gateway:0.0.1

cmd /k
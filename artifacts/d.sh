#!/bin/bash

echo eval $(minikube docker-env)
eval eval $(minikube docker-env)

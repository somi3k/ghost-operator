#!/bin/bash

kubectl delete deploy --all
kubectl delete service --all
kubectl delete pvc --all
kubectl delete crd --all
kubectl delete pv --all

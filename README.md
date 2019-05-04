# ghost-operator

## ghost-operator is a Kubernetes API extension consisting of a Custom Resource Definition and Custom Controller following the Operator pattern.  The goal of this project is to create an operator which can be deployed on a Kubernetes cluster, allowing the user to create and manage Ghost instances via the kubectl command line interface.  This ghost-operator is cloned from k8s.io/sample-controller and is UNDER DEVELOPMENT.  It will be updated incrementally and this Readme will reflect current progress on this project.

<br>

## Installation and Dependencies
1. the Kubernetes cluster version should be greater than 1.9.
2. clone this repository <code> git clone https://github.com/somi3k/ghost-operator </code>
3. if running on minikube, set the docker daemon environment <code> eval $(minikube docker-env) </code>
4. build the ghost-controller <code> export GOOS=linux; go build . </code>
5. move the ghost-controller binary to ./artifacts <code> mv ghost-controller ./artifacts/ </code>
6. build the ghost-operator container <code> docker build -t ghost-operator:v1alpha1 ./artifacts/ </code>

<br>

## Running ghost-operator
1. deploy crd, rbac and controller <code> kubectl apply -f ./artifacts/deploy-ghost-operator.yaml </code>
2. deploy service, persistent volume and pvc <code> kubectl apply -f ./artifacts/deploy-ghost.yaml </code>
3. modify <code> /artifacts/ghost.yaml </code> to match the desired installation specs
4. deploy ghost container <code> kubectl apply -f ./arifacts/ghost.yaml </code>

<br>

## Using ghost-operator
1. view running Ghost deployments <code> kubectl get ghosts </code>
2. view full deployment configuration <code> kubectl describe ghost <deployment name>  </code>
3. delete Ghost deployment <code> kubectl delete ghost <deployment name> </code>

<br>

### ghost-operator is *under development* and this readme will be updated as additional features are implemented.



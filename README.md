# ghost-operator

*ghost-operator* is a Kubernetes API extension consisting of a Custom Resource Definition and Custom Controller following the Operator pattern.  The goal of this project is to create an operator which can be deployed on a Kubernetes cluster, allowing the user to create and manage Ghost instances via the kubectl command line interface.  This ghost-operator is based on the sample-controller cloned from https://github.com/kubernetes/sample-controller and will be updated incrementally as features are added.  This Readme will reflect the current state of this project.

<br>


## Running ghost-operator
1. the Kubernetes cluster version should be greater than 1.9.
2. clone this repo <code> git clone https<span></span>://github.com/somi3k/ghost-operator </code>
3. deploy ghost controller <code> kubectl apply -f ./artifacts/deploy-ghost-operator.yaml </code>
4. modify <code> /artifacts/ghost.yaml </code> to reflect the desired deployment name
5. deploy ghost container <code> kubectl apply -f ./artifacts/ghost.yaml </code>
<br>

![ghost-operator setup](https://raw.githubusercontent.com/somi3k/ghost-operator/master/images/deploy.jpg)
<br>


## Using ghost-operator
1. view running Ghost deployment <code> kubectl get ghosts </code>
2. view full deployment configuration <code> kubectl describe ghost [deployment name]  </code>
3. delete Ghost deployment <code> kubectl delete ghost [deployment name] </code>
<br>

![ghost-operator status](https://raw.githubusercontent.com/somi3k/ghost-operator/master/images/status.jpg)
<br>


## Modifying Installation Locally
1. the Kubernetes cluster version should be greater than 1.9.
2. clone this repo <code> git clone https<span></span>://github.com/somi3k/ghost-operator </code>
3. install dep <code> sudo apt-get install go-dep </code>
4. install project dependencies <code> dep ensure </code>
5. run <code> ./hack/update-codegen.sh </code> after making changes to project files
5. if running on minikube, set the docker daemon <code> eval $(minikube docker-env) </code>
6. build the ghost-controller <code> export GOOS=linux; go build . </code> *GOOS=darwin for Macs*
7. move the ghost-controller binary <code> mv ghost-controller ./artifacts/ </code>
8. build the ghost-operator container <code> docker build -t ghost-operator:v1alpha1 ./artifacts/ </code>

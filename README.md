# Software Network Analysis Gateway

This component is a microservice built for the system detailed in the [infrastructure repo](https://github.com/ryanlaycock/software-network-analysis-infrastructure/tree/develop).
The docker container can be pulled from [here](https://hub.docker.com/r/ryanlaycock/software-network-analysis-gateway).

Social Network Analysis is a REST API, written in Golang that [exposes endpoints](software-network-analysis-gateway.yaml) 
to retrieve software complexity and dependency metrics for Java Maven projects. After analysing projects, the results
are cached, making future lookup faster.

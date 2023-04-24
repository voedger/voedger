## Swarm mode key concepts

https://docs.docker.com/engine/swarm/key-concepts/

- A **Node** is an instance of the Docker engine participating in the swarm
- A **Service** is the definition of the tasks to execute on the manager or worker nodes
  - When you create a service, you specify which container image to use and which commands to execute inside running containers.
- A **Task** carries a Docker container and the commands to run inside the container. It is the atomic scheduling unit of swarm
  - Manager nodes assign tasks to worker nodes according to the number of replicas set in the service scale. 
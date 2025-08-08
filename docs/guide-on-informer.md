## The Architecture of Controllers
Since controllers are in charge of meeting the desired state of the resources in Kubernetes, they somehow need to be informed about the changes on the resources and perform certain operations if needed. For this, controllers follow a special architecture to

1) observe the resources,
2) inform any events (updating, deleting, adding) done on the resources,
3) keep a local cache to decrease the load on API Server,
4) keep a work queue to pick up events,
5) run workers to perform reconciliation on resources picked up from work queue.

 <a href="https://www.nakamasato.com/kubernetes-training/kubernetes-operator/client-go/informer/" >
    <h4 class="text-yellow-300 text-lg">Ref: Official Docs</h4>
    </a>
 <a href="https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md">
    <h4 class="text-yellow-300 text-lg">Writing Controllers (Imp)</h4>
    </a>

# Factory & Informers

![image](https://github.com/user-attachments/assets/bb09fdaf-a1d8-4f9b-bfd4-a7914ebe6eba)

# Single Informer

![image](https://github.com/user-attachments/assets/bfc1720a-aab4-4595-bb4b-9559a3e98d74)

![image](https://github.com/user-attachments/assets/1af2d969-5b93-40f4-b375-219342c16041)


[//]: # (https://github.com/user-attachments/assets/851d2b2b-d268-4894-a15a-dbe8b501b3cc)

## Definition : Informer

Informer monitors the changes of target resource. An informer is created for each of the target resources if you need to handle multiple resources (e.g. podInformer, deploymentInformer).



```md
1) Initialize the Controller
	* The NewController function sets up the Kubernetes controller with a work queue, informer, and WebSocket connection.
	* It listens for Deployment events (Add, Update, Delete) and enqueues them.

2) Run the Controller
	* The Run method waits for cache synchronization and starts the worker loop.
	* It continuously processes events from the work queue.

3) Process Deployment Events
	* The processItem method retrieves Deployment events from the queue and determines the necessary action.
	* It fetches the Deployment details and handles errors, deletions, and updates.

4) Handle Deployment Changes
	* `handleAdd`, `handleUpdate`, and `handleDel` respond to Deployment changes.
	* Updates track Replica count and Image changes and send logs via WebSocket.

5) Send Updates via WebSocket
	* The updateLogs function logs and sends JSON messages about Deployment changes.
	* The WebSocket connection ensures real-time updates for external systems.

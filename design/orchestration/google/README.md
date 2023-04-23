## Abstract

This paper overviews the following documents:

- [Borg, Omega, and Kubernetes][BOK], Brendan Burns Brian Grant David Oppenheimer Eric Brewer John Wilkes, ACM Queue, vol. 14 (2016), pp. 70-93
- [Omega: flexible, scalable schedulers for large compute clusters][Omega], Malte Schwarzkopf Andy Konwinski Michael Abd-El-Malek John Wilkes, SIGOPS European Conference on Computer Systems (EuroSys), ACM, Prague, Czech Republic (2013), pp. 351-364
- [Large-scale cluster management at Google with Borg][Borg], Abhishek Verma Luis Pedrosa Madhukar R. Korupolu David Oppenheimer Eric Tune John Wilkes
Proceedings of the European Conference on Computer Systems (EuroSys), ACM, Bordeaux, France (2015)

Takeaways:
 
1. The scheduler must use a *shared state* model with *optimistic* concurrency control without blocking resources while scheduling jobs.
2. The scheduler must support different scheduling algorithms for *different types of jobs* (service and batch), each of which fulfills a minimum set of common requirements (for example, has a common scale of priority of jobs or common metrics of maximum resources).
3. The default scheduler should schedule jobs *incrementally*, but for individual jobs, allow all-or-nothing scheduling.
4. Сoncurrency conflicts resolution should not use "outdated scheduling shapshot" as a reason, analyse results instead (reason: decrease conflicts frequence)
5. The *scaling* of the scheduler under the increasing load on the job queue should occur, if possible, automatically or by simply changing the configuration of the scheduler, without rewriting its source codes.


## The History of Kubernetes on a Timeline

https://blog.risingstack.com/the-history-of-kubernetes
- 2003-2004: Birth of the Borg System
- 2013: From Borg to Omega
- 2014: Google Introduces Kubernetes
- 2016: The Year Kubernetes Goes Mainstream!
- 2017: The Year of Enterprise Adoption & Support


## Borg, Omega, and Kubernetes

[Borg, Omega, and Kubernetes][BOK]

- Though widespread interest in software containers is a relatively recent phenomenon, at Google we have been managing Linux containers at scale for more than ten years and built three different containermanagement systems in that time.
- Each system was heavily influenced by its predecessors, even though they were developed for different reasons

### Borg

- The first unified container-management system developed at Google was the system we internally call Borg
- Borg remains the primary container-management system within Google because of its scale, breadth of features, and extreme robustness

### Omega

- Omega an offspring of Borg, was driven by a desire to improve the software engineering of the Borg ecosystem.
- It applied many of the patterns that had proved successful in Borg, but was built from the ground up to have a more consistent, principled architecture
- Omega stored the state of the cluster in a centralized Paxos-based transaction oriented store that was accessed by the different parts of the cluster control plane (such as schedulers), using optimistic concurrency control to handle the occasional conflicts.
- Many of Omega’s innovations (including multiple schedulers) have since been folded into Borg

### Kubernetes

- The third container-management system developed at Google was Kubernetes
-  It was conceived of and developed in a world where external developers were becoming
interested in Linux containers, and Google had developed a growing business selling public-cloud infrastructure.
- Kubernetes is open source—a contrast to Borg and Omega, which were developed as purely Google-internal systems.
- Like Omega, Kubernetes has at its core a shared persistent store, with components watching for changes to relevant objects
- In contrast to Omega, which exposes the store directly to trusted control-plane components, state in Kubernetes is accessed exclusively through a domains pecific REST API that applies higher-level versioning, validation, semantics, and policy, in support of a more diverse array of clients


## Omega: flexible, scalable schedulers for large compute clusters

- [Omega: flexible, scalable schedulers for large compute clusters][Omega]
- Russian translation: [Omega-ru](Omega-ru.md)



## Large-scale cluster management at Google with Borg

- [Large-scale cluster management at Google with Borg][Borg]
- To be read


[Omega]: https://research.google/pubs/pub41684/
[Borg]: https://research.google/pubs/pub43438/
[BOK]: https://research.google/pubs/pub44843/



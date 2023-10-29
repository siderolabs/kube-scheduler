# kube-scheduler

# Deploying

- Create a WattTime account: https://www.watttime.org/api-documentation/#register-new-user
- Create `PriorityClass`
- Annotate nodes with:
  - `bmc.siderolabs.com/endpoint`
  - `bmc.siderolabs.com/username`
  - `bmc.siderolabs.com/password`
- Deploy the scheduler
- Create pod with `priorityClassName` referencing the `PriorityClass` created above
- Create pod with `schedulerName` set to `kube-scheduler-siderolabs`

# Logic

- Evict pods with `priority` < `index`
- Power off nodes when idle AND no pods are in the queue (pending) with `priority` >= `index`
- Power on nodes when pods are in the queue (pending) with `priority` >= `index`

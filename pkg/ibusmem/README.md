### Limitations

- Sender context cancellation is seen by the Receiver only after call to `sectionsWriter()`
  - Reason: the more cases there are in select the slower things are (seems ~200 ns per case)



### Perfomance

See [perfomance.md](perfomance.md) for more information.
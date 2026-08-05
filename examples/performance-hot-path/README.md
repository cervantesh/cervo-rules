# Performance Hot Path

This example shows the recommended shape for latency-sensitive consumers:

- compile the policy once at startup;
- compile the HTTP classifier once at startup;
- use `DecideWithOptions` when trace and observation are not needed;
- prepare stable facts outside the request path and evaluate only request facts
  with explicit budgets.

The example is intentionally neutral: it routes billing operations and does not
own any transport or backend behavior beyond local request classification.

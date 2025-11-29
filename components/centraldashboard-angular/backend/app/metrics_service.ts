/** Time-series interval enumeration. */
export enum Interval {
  Last5m,
  Last15m,
  Last30m,
  Last60m,
  Last180m
}

/** Data-point contained in a time series. */
export interface TimeSeriesPoint {
  timestamp: number;
  label: string;
  value: number;
}

/**
 * Interface definition for implementers of metrics services capable of
 * returning time-series resource utilization metrics for the Kubeflow system.
 */
export interface MetricsService {
  /**
   * Nodes in the cluster.
   * @param interval
   */
  getNodeCpuUtilization(interval: Interval): Promise<TimeSeriesPoint[]>;

  /**
   * within the cluster.
   * @param interval
   */
  getPodCpuUtilization(interval: Interval): Promise<TimeSeriesPoint[]>;

  /**
   * for Pods within the cluster.
   * @param interval
   */
  getPodMemoryUsage(interval: Interval): Promise<TimeSeriesPoint[]>;
}

import React, { useEffect, useState } from 'react';
import { metricsUsageApi } from '../services/api';
import { Button } from './ui';

interface LabelCardinality {
  [key: string]: number;
}

interface MetricUsageInfo {
  metric_name: string;
  sample_count: number;
  first_seen: string;
  last_seen: string;
  cardinality: number;
  label_cardinality: LabelCardinality;
  min_value: number;
  max_value: number;
  sum_value: number;
  avg_value: number;
}

interface MetricsUsageResponse {
  metrics: MetricUsageInfo[];
  total: number;
}

const MetricsUsagePanel: React.FC = () => {
  const [metricsUsage, setMetricsUsage] = useState<MetricUsageInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedMetric, setSelectedMetric] = useState<MetricUsageInfo | null>(null);
  const [sortBy, setSortBy] = useState<string>('cardinality');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  const fetchMetricsUsage = async () => {
    setRefreshing(true);
    setError(null);
    
    try {
      const response = await metricsUsageApi.getAllMetricsUsage();
      
      if (response.error) {
        setError(response.error);
      } else if (response.data) {
        const data = response.data as unknown as MetricsUsageResponse;
        setMetricsUsage(data.metrics);
      }
    } catch (err) {
      setError(`Failed to fetch metrics usage: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };
  
  useEffect(() => {
    fetchMetricsUsage();
  }, []);
  
  const handleRefresh = () => {
    fetchMetricsUsage();
  };
  
  const handleSort = (column: string) => {
    if (sortBy === column) {
      // Toggle sort order if clicking the same column
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      // Default to descending for new column
      setSortBy(column);
      setSortOrder('desc');
    }
  };
  
  const sortedMetrics = [...metricsUsage].sort((a, b) => {
    let comparison = 0;
    
    switch (sortBy) {
      case 'metric_name':
        comparison = a.metric_name.localeCompare(b.metric_name);
        break;
      case 'cardinality':
        comparison = a.cardinality - b.cardinality;
        break;
      case 'sample_count':
        comparison = a.sample_count - b.sample_count;
        break;
      case 'last_seen':
        comparison = new Date(a.last_seen).getTime() - new Date(b.last_seen).getTime();
        break;
      default:
        comparison = 0;
    }
    
    return sortOrder === 'asc' ? comparison : -comparison;
  });
  
  return (
    <div className="p-4">
      <div className="mb-6 flex justify-between items-center">
        <h2 className="text-2xl font-bold">Metrics Usage</h2>
        <Button
          onClick={handleRefresh}
          disabled={refreshing}
          variant="primary"
        >
          {refreshing ? 'Refreshing...' : 'Refresh'}
        </Button>
      </div>
      
      {error && (
        <div className="mb-4 p-3 bg-red-100 text-red-700 rounded">
          {error}
        </div>
      )}
      
      {loading ? (
        <div className="text-center py-8">
          <p>Loading metrics usage data...</p>
        </div>
      ) : (
        <>
          {metricsUsage.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-600">No metrics usage data available yet.</p>
              <p className="mt-2 text-sm text-gray-500">
                Metrics are tracked as they are processed by the system.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full bg-white border border-gray-200">
                <thead>
                  <tr className="bg-gray-100">
                    <th 
                      className="px-4 py-2 text-left cursor-pointer" 
                      onClick={() => handleSort('metric_name')}
                    >
                      Metric Name {sortBy === 'metric_name' && (sortOrder === 'asc' ? '↑' : '↓')}
                    </th>
                    <th 
                      className="px-4 py-2 text-right cursor-pointer" 
                      onClick={() => handleSort('cardinality')}
                    >
                      Cardinality {sortBy === 'cardinality' && (sortOrder === 'asc' ? '↑' : '↓')}
                    </th>
                    <th 
                      className="px-4 py-2 text-right cursor-pointer" 
                      onClick={() => handleSort('sample_count')}
                    >
                      Samples {sortBy === 'sample_count' && (sortOrder === 'asc' ? '↑' : '↓')}
                    </th>
                    <th className="px-4 py-2 text-right">Min</th>
                    <th className="px-4 py-2 text-right">Max</th>
                    <th className="px-4 py-2 text-right">Avg</th>
                    <th 
                      className="px-4 py-2 text-right cursor-pointer" 
                      onClick={() => handleSort('last_seen')}
                    >
                      Last Seen {sortBy === 'last_seen' && (sortOrder === 'asc' ? '↑' : '↓')}
                    </th>
                    <th className="px-4 py-2 text-center">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedMetrics.map((metric) => (
                    <tr 
                      key={metric.metric_name} 
                      className="border-t border-gray-200 hover:bg-gray-50"
                    >
                      <td className="px-4 py-2 font-medium">{metric.metric_name}</td>
                      <td className="px-4 py-2 text-right">{metric.cardinality.toLocaleString()}</td>
                      <td className="px-4 py-2 text-right">{metric.sample_count.toLocaleString()}</td>
                      <td className="px-4 py-2 text-right">{metric.min_value.toLocaleString(undefined, { maximumFractionDigits: 2 })}</td>
                      <td className="px-4 py-2 text-right">{metric.max_value.toLocaleString(undefined, { maximumFractionDigits: 2 })}</td>
                      <td className="px-4 py-2 text-right">{metric.avg_value.toLocaleString(undefined, { maximumFractionDigits: 2 })}</td>
                      <td className="px-4 py-2 text-right">{new Date(metric.last_seen).toLocaleString()}</td>
                      <td className="px-4 py-2 text-center">
                        <button 
                          className="text-blue-500 hover:text-blue-700"
                          onClick={() => setSelectedMetric(metric)}
                        >
                          Details
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          
          {selectedMetric && (
            <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
              <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full max-h-screen overflow-y-auto">
                <div className="p-6">
                  <div className="flex justify-between items-start mb-4">
                    <h3 className="text-xl font-bold">{selectedMetric.metric_name}</h3>
                    <button 
                      className="text-gray-500 hover:text-gray-700"
                      onClick={() => setSelectedMetric(null)}
                    >
                      ✕
                    </button>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4 mb-4">
                    <div>
                      <p className="text-sm text-gray-500">First Seen</p>
                      <p>{new Date(selectedMetric.first_seen).toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Last Seen</p>
                      <p>{new Date(selectedMetric.last_seen).toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Sample Count</p>
                      <p>{selectedMetric.sample_count.toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Cardinality</p>
                      <p>{selectedMetric.cardinality.toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Min Value</p>
                      <p>{selectedMetric.min_value.toLocaleString(undefined, { maximumFractionDigits: 4 })}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Max Value</p>
                      <p>{selectedMetric.max_value.toLocaleString(undefined, { maximumFractionDigits: 4 })}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Sum Value</p>
                      <p>{selectedMetric.sum_value.toLocaleString(undefined, { maximumFractionDigits: 4 })}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-500">Average Value</p>
                      <p>{selectedMetric.avg_value.toLocaleString(undefined, { maximumFractionDigits: 4 })}</p>
                    </div>
                  </div>
                  
                  <div className="mb-4">
                    <h4 className="font-medium mb-2">Label Cardinality</h4>
                    {Object.keys(selectedMetric.label_cardinality).length === 0 ? (
                      <p className="text-gray-500">No labels tracked for this metric</p>
                    ) : (
                      <table className="min-w-full bg-white border border-gray-200">
                        <thead>
                          <tr className="bg-gray-100">
                            <th className="px-4 py-2 text-left">Label</th>
                            <th className="px-4 py-2 text-right">Unique Values</th>
                          </tr>
                        </thead>
                        <tbody>
                          {Object.entries(selectedMetric.label_cardinality)
                            .sort(([, a], [, b]) => b - a)
                            .map(([label, cardinality]) => (
                              <tr key={label} className="border-t border-gray-200">
                                <td className="px-4 py-2 font-mono text-sm">{label}</td>
                                <td className="px-4 py-2 text-right">{cardinality.toLocaleString()}</td>
                              </tr>
                            ))}
                        </tbody>
                      </table>
                    )}
                  </div>
                  
                  <div className="flex justify-end">
                    <Button 
                      variant="secondary" 
                      onClick={() => setSelectedMetric(null)}
                    >
                      Close
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default MetricsUsagePanel;
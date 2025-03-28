import React, { useState, useEffect } from 'react';
import { rulesApi } from '../services/api';
import { Button } from './ui';

interface RuleFormData {
  id?: string;
  name: string;
  description: string;
  sourceMetricsInput: string; // Use a simple string for input
  aggregationType: string;
  active: boolean;
  labelsToKeep: string[];
  outputMetricName: string;
  dropOriginalMetric?: boolean;
}

interface RuleFormProps {
  ruleId?: string;
  onSave?: () => void;
  onCancel?: () => void;
  onClose?: () => void; // Add onClose prop for compatibility
}

const aggregationTypes = [
  { value: 'sum', label: 'Sum' },
  { value: 'avg', label: 'Average' },
  { value: 'min', label: 'Minimum' },
  { value: 'max', label: 'Maximum' },
  { value: 'count', label: 'Count' }
];

const RuleForm: React.FC<RuleFormProps> = ({ ruleId, onSave, onCancel, onClose }) => {
  const [formData, setFormData] = useState<RuleFormData>({
    name: '',
    description: '',
    sourceMetricsInput: '', // Initialize as an empty string
    aggregationType: 'sum',
    active: true,
    labelsToKeep: [],
    outputMetricName: '',
    dropOriginalMetric: false
  });
  
  const [loading, setLoading] = useState<boolean>(false);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [newLabelKey, setNewLabelKey] = useState<string>('');
  const [addingLabel, setAddingLabel] = useState<boolean>(false);
  
  // Fetch rule details if editing an existing rule
  useEffect(() => {
    if (ruleId) {
      const fetchRule = async () => {
        setLoading(true);
        setError(null);
        
        try {
          const response = await rulesApi.getRule(ruleId);
          
          if (response.error) {
            setError(response.error);
          } else if (response.data) {
            // Convert the rule data to form format
            const sourceMetrics = response.data.matcher?.metric_names || 
                                 (response.data.sourceMetric ? [response.data.sourceMetric] : []);
            
            setFormData({
              id: response.data.id,
              name: response.data.name || '',
              description: response.data.description || '',
              sourceMetricsInput: sourceMetrics.join(','), // Convert array to comma-separated string
              aggregationType: response.data.aggregationType || 'sum',
              active: response.data.enabled !== false,
              labelsToKeep: response.data.aggregation?.segmentation || [],
              outputMetricName: response.data.output?.metric_name || '',
              dropOriginalMetric: response.data.output?.drop_original || false
            });
          }
        } catch (err) {
          setError(`Failed to fetch rule: ${err instanceof Error ? err.message : String(err)}`);
        } finally {
          setLoading(false);
        }
      };
      
      fetchRule();
    }
  }, [ruleId]);
  
  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    const checked = type === 'checkbox' ? (e.target as HTMLInputElement).checked : undefined;
    
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };
  
  const handleAddLabel = async () => {
    if (newLabelKey && !formData.labelsToKeep.includes(newLabelKey)) {
      setAddingLabel(true);
      try {
        setFormData(prev => ({
          ...prev,
          labelsToKeep: [...prev.labelsToKeep, newLabelKey]
        }));
        setNewLabelKey('');
      } finally {
        setAddingLabel(false);
      }
    }
  };
  
  const handleRemoveLabel = (label: string) => {
    setFormData(prev => ({
      ...prev,
      labelsToKeep: prev.labelsToKeep.filter(l => l !== label)
    }));
  };
  
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    
    // Validate that sourceMetricsInput is not empty
    if (!formData.sourceMetricsInput.trim()) {
      setError("At least one metric name must be specified");
      setSaving(false);
      return;
    }
    
    try {
      // Transform the form data to match the expected backend structure
      const ruleData = {
        id: formData.id,
        name: formData.name,
        description: formData.description,
        enabled: formData.active,
        matcher: {
          metric_names: formData.sourceMetricsInput.split(',').map(m => m.trim()).filter(Boolean), // Convert string to array
          labels: {},
          label_regex: {}
        },
        aggregation: {
          type: formData.aggregationType,
          interval_seconds: 60,  // Default value
          segmentation: formData.labelsToKeep
        },
        output: {
          metric_name: formData.outputMetricName,
          drop_original: formData.dropOriginalMetric,
          keep_labels: formData.labelsToKeep
        }
      };
      
      let response;
      
      if (ruleId) {
        // Update existing rule
        response = await rulesApi.updateRule(ruleId, ruleData);
      } else {
        // Create new rule
        response = await rulesApi.createRule(ruleData);
      }
      
      if (response.error) {
        setError(response.error);
      } else {
        if (onSave) {
          onSave();
        }
        // Call onClose after successful save if provided
        if (onClose) {
          onClose();
        }
      }
    } catch (err) {
      setError(`Failed to save rule: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setSaving(false);
    }
  };
  
  if (loading) {
    return (
      <div className="flex justify-center items-center h-32">
        <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    );
  }
  
  return (
    <div className="bg-white shadow rounded-lg p-6">
      <h2 className="text-xl font-semibold text-gray-800 mb-4">
        {ruleId ? 'Edit Aggregation Rule' : 'Create New Aggregation Rule'}
      </h2>
      
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}
      
      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
              Rule Name *
            </label>
            <input
              type="text"
              id="name"
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter rule name"
            />
          </div>
          
          <div>
            <label htmlFor="sourceMetricsInput" className="block text-sm font-medium text-gray-700 mb-1">
              Source Metrics *
            </label>
            <input
              type="text"
              id="sourceMetricsInput"
              name="sourceMetricsInput"
              value={formData.sourceMetricsInput}
              onChange={handleChange}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter source metric names separated by commas"
            />
          </div>
        </div>
        
        <div className="mb-4">
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
            Description
          </label>
          <textarea
            id="description"
            name="description"
            value={formData.description}
            onChange={handleChange}
            rows={2}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Enter rule description"
          />
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label htmlFor="aggregationType" className="block text-sm font-medium text-gray-700 mb-1">
              Aggregation Type *
            </label>
            <select
              id="aggregationType"
              name="aggregationType"
              value={formData.aggregationType}
              onChange={handleChange}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {aggregationTypes.map(type => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>
          
          <div>
            <label htmlFor="outputMetricName" className="block text-sm font-medium text-gray-700 mb-1">
              Output Metric Name *
            </label>
            <input
              type="text"
              id="outputMetricName"
              name="outputMetricName"
              value={formData.outputMetricName}
              onChange={handleChange}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter output metric name"
            />
          </div>
        </div>
        
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Labels to Keep
          </label>
          <div className="flex flex-wrap gap-2 mb-2">
            {formData.labelsToKeep.map(label => (
              <span 
                key={label}
                className="inline-flex items-center bg-gray-100 text-gray-800 px-2 py-1 rounded-md text-sm"
              >
                {label}
                <button 
                  type="button"
                  onClick={() => handleRemoveLabel(label)}
                  className="ml-1 text-gray-500 hover:text-red-500"
                >
                  ×
                </button>
              </span>
            ))}
          </div>
          <div className="flex">
            <input
              type="text"
              value={newLabelKey}
              onChange={(e) => setNewLabelKey(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-l-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Add label key"
            />
            <Button
              type="button"
              onClick={handleAddLabel}
              isLoading={addingLabel}
              className="rounded-l-none rounded-r-md"
            >
              Add
            </Button>
          </div>
        </div>
        
        <div className="flex items-center mb-4">
          <input
            type="checkbox"
            id="active"
            name="active"
            checked={formData.active}
            onChange={handleChange}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <label htmlFor="active" className="ml-2 text-sm text-gray-700">
            Active (Enable this rule for processing)
          </label>
        </div>
        
        <div className="flex items-center mb-6">
          <input
            type="checkbox"
            id="dropOriginalMetric"
            name="dropOriginalMetric"
            checked={formData.dropOriginalMetric}
            onChange={handleChange}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <label htmlFor="dropOriginalMetric" className="ml-2 text-sm text-gray-700">
            Drop Original Metric (Remove the source metric after aggregation)
          </label>
        </div>
        
        <div className="flex justify-end gap-3">
          <Button
            type="button"
            onClick={onClose || onCancel}
            disabled={saving}
            variant="secondary"
            isLoading={false}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            disabled={saving}
            isLoading={saving}
            loadingText={ruleId ? 'Updating...' : 'Creating...'}
          >
            {ruleId ? 'Update Rule' : 'Create Rule'}
          </Button>
        </div>
      </form>
    </div>
  );
};

export default RuleForm;
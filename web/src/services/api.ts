import { ApiResponse } from '../types/metrics';

// Base API URL - adjust based on your environment
// During development, point to the Go backend server
const API_BASE_URL = 'http://localhost:8080/api';

// Utility function for handling API responses
const handleResponse = async <T>(response: Response): Promise<ApiResponse<T>> => {
  if (!response.ok) {
    const errorText = await response.text();
    return { error: errorText || `Error: ${response.status} ${response.statusText}` };
  }
  
  // Handle 204 No Content responses specially
  if (response.status === 204) {
    return { data: null as unknown as T };
  }
  
  try {
    const data = await response.json();
    return { data };
  } catch (error) {
    return { error: 'Invalid JSON response' };
  }
};

// Rule management API
export const rulesApi = {
  // Get all rules
  getAllRules: async (): Promise<ApiResponse<any[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/rules`);
      return handleResponse<any[]>(response);
    } catch (error) {
      return { error: `Failed to fetch rules: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Get a specific rule
  getRule: async (id: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/rules/${encodeURIComponent(id)}`);
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to fetch rule ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Create a new rule
  createRule: async (ruleData: any): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/rules`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(ruleData),
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to create rule: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Update an existing rule
  updateRule: async (id: string, ruleData: any): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/rules/${encodeURIComponent(id)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(ruleData),
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to update rule ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Delete a rule
  deleteRule: async (id: string): Promise<ApiResponse<void>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/rules/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
      return handleResponse<void>(response);
    } catch (error) {
      return { error: `Failed to delete rule ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  }
};

// Recommendations API
export const recommendationsApi = {
  // Get all recommendations
  getAllRecommendations: async (): Promise<ApiResponse<any[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/recommendations`);
      return handleResponse<any[]>(response);
    } catch (error) {
      return { error: `Failed to fetch recommendations: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Get a specific recommendation
  getRecommendation: async (id: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/recommendations/${encodeURIComponent(id)}`);
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to fetch recommendation ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Apply a recommendation
  applyRecommendation: async (id: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/recommendations/${encodeURIComponent(id)}/apply`, {
        method: 'POST',
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to apply recommendation ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Reject a recommendation
  rejectRecommendation: async (id: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/recommendations/${encodeURIComponent(id)}/reject`, {
        method: 'POST',
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to reject recommendation ${id}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Generate new recommendations
  generateRecommendations: async (): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/recommendations/generate`, {
        method: 'POST',
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to generate recommendations: ${error instanceof Error ? error.message : String(error)}` };
    }
  }
};

// Kubernetes integration API
export const kubernetesApi = {
  // Get Kubernetes monitor YAML for a rule
  getMonitor: async (ruleId: string): Promise<ApiResponse<string>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/kubernetes/monitors/${encodeURIComponent(ruleId)}`);
      
      // Special handling for YAML response
      if (!response.ok) {
        const errorText = await response.text();
        return { error: errorText || `Error: ${response.status} ${response.statusText}` };
      }
      
      const yamlText = await response.text();
      return { data: yamlText };
    } catch (error) {
      return { error: `Failed to get Kubernetes monitor for rule ${ruleId}: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Save Kubernetes monitor to a file
  saveMonitor: async (ruleId: string, outputDir?: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/kubernetes/monitors/${encodeURIComponent(ruleId)}/save`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ output_dir: outputDir }),
      });
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to save Kubernetes monitor for rule ${ruleId}: ${error instanceof Error ? error.message : String(error)}` };
    }
  }
};

// Health check API
export const systemApi = {
  getHealth: async (): Promise<ApiResponse<any>> => {
    try {
      // The health endpoint is at the root level, not under the API path
      const response = await fetch(`http://localhost:8080/health`);
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to get system health: ${error instanceof Error ? error.message : String(error)}` };
    }
  }
};

// Metrics usage API
export const metricsUsageApi = {
  // Get all metrics usage info
  getAllMetricsUsage: async (): Promise<ApiResponse<any[]>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/metrics-usage`);
      return handleResponse<any[]>(response);
    } catch (error) {
      return { error: `Failed to fetch metrics usage: ${error instanceof Error ? error.message : String(error)}` };
    }
  },
  
  // Get a specific metric's usage info
  getMetricUsage: async (name: string): Promise<ApiResponse<any>> => {
    try {
      const response = await fetch(`${API_BASE_URL}/v1/metrics-usage/${encodeURIComponent(name)}`);
      return handleResponse<any>(response);
    } catch (error) {
      return { error: `Failed to fetch usage for metric ${name}: ${error instanceof Error ? error.message : String(error)}` };
    }
  }
};
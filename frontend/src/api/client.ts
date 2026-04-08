const API_BASE = import.meta.env.VITE_API_BASE_URL || import.meta.env.VITE_API_BASE || 'http://localhost:8080';

export interface User {
  id: string;
  email: string;
  created_at: string;
}

export type WorkloadType = 'web_service' | 'static_site';

export interface Project {
  id: string;
  user_id: string;
  name: string;
  git_repo_url: string;
  branch: string;
  workload_type: WorkloadType;
  slug?: string;
  created_at: string;
  updated_at: string;
}

export interface ProjectConfig {
  project_id: string;
  build_method: string;
  build_cmd?: string | null;
  start_cmd?: string | null;
  dockerfile_path?: string | null;
  output_dir?: string | null;
  install_cmd?: string | null;
  port?: number | null;
  healthcheck_path?: string | null;
  env_vars?: Record<string, any> | null;
  created_at: string;
  updated_at: string;
}

export type DeploymentStatus = 
  | 'queued' 
  | 'building' 
  | 'build_failed' 
  | 'deploying' 
  | 'running' 
  | 'failed' 
  | 'stopped' 
  | 'rolled_back';

export interface Deployment {
  id: string;
  project_id: string;
  git_commit_sha?: string;
  git_branch?: string;
  status: DeploymentStatus;
  source_type?: string;
  artifact_path?: string;
  runtime_container_id?: string;
  public_url?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  created_by_user_id?: string;
}

export interface DeploymentEvent {
  id: string;
  deployment_id: string;
  event_type: string;
  message: string;
  timestamp: string;
  metadata_json?: any;
}

interface AuthResponse {
  user: User;
}

interface ProjectResponse {
  project: Project;
}

interface ProjectListResponse {
  projects: Project[];
}

interface ProjectConfigResponse {
  config: ProjectConfig;
}

interface DeploymentResponse {
  deployment: Deployment;
}

interface DeploymentListResponse {
  deployments: Deployment[];
}

interface EventListResponse {
  events: DeploymentEvent[];
}

class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'APIError';
  }
}

async function fetchJSON<T>(url: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new APIError(response.status, error.error || 'Request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export const api = {
  auth: {
    signup: async (email: string, password: string): Promise<User> => {
      const response = await fetchJSON<AuthResponse>('/auth/signup', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      return response.user;
    },

    login: async (email: string, password: string): Promise<User> => {
      const response = await fetchJSON<AuthResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      return response.user;
    },

    logout: async (): Promise<void> => {
      await fetchJSON<void>('/auth/logout', { method: 'POST' });
    },

    me: async (): Promise<User | null> => {
      try {
        const response = await fetchJSON<AuthResponse>('/auth/me');
        return response.user;
      } catch (error) {
        if (error instanceof APIError && error.status === 401) {
          return null;
        }
        throw error;
      }
    },
  },

  projects: {
    list: async (): Promise<Project[]> => {
      const response = await fetchJSON<ProjectListResponse>('/projects');
      return response.projects;
    },

    get: async (id: string): Promise<Project> => {
      const response = await fetchJSON<ProjectResponse>(`/projects/${id}`);
      return response.project;
    },

    create: async (name: string, gitRepoUrl: string, branch: string, workloadType: WorkloadType): Promise<Project> => {
      const response = await fetchJSON<ProjectResponse>('/projects', {
        method: 'POST',
        body: JSON.stringify({ 
          name, 
          git_repo_url: gitRepoUrl, 
          branch,
          workload_type: workloadType
        }),
      });
      return response.project;
    },

    update: async (
      id: string,
      name: string,
      gitRepoUrl: string,
      branch: string,
      workloadType: WorkloadType
    ): Promise<Project> => {
      const response = await fetchJSON<ProjectResponse>(`/projects/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ 
          name, 
          git_repo_url: gitRepoUrl, 
          branch,
          workload_type: workloadType
        }),
      });
      return response.project;
    },

    delete: async (id: string): Promise<void> => {
      await fetchJSON<void>(`/projects/${id}`, { method: 'DELETE' });
    },

    getConfig: async (id: string): Promise<ProjectConfig> => {
      const response = await fetchJSON<ProjectConfigResponse>(`/projects/${id}/config`);
      return response.config;
    },

    updateConfig: async (id: string, config: Partial<ProjectConfig>): Promise<ProjectConfig> => {
      const response = await fetchJSON<ProjectConfigResponse>(`/projects/${id}/config`, {
        method: 'PUT',
        body: JSON.stringify(config),
      });
      return response.config;
    },

    deploy: async (id: string, commitSha?: string, branch?: string): Promise<Deployment> => {
      const response = await fetchJSON<DeploymentResponse>(`/projects/${id}/deploy`, {
        method: 'POST',
        body: JSON.stringify({ commit_sha: commitSha, branch }),
      });
      return response.deployment;
    },

    listDeployments: async (id: string): Promise<Deployment[]> => {
      const response = await fetchJSON<DeploymentListResponse>(`/projects/${id}/deployments`);
      return response.deployments;
    },
  },

  deployments: {
    get: async (id: string): Promise<Deployment> => {
      const response = await fetchJSON<DeploymentResponse>(`/deployments/${id}`);
      return response.deployment;
    },

    listEvents: async (id: string): Promise<DeploymentEvent[]> => {
      const response = await fetchJSON<EventListResponse>(`/deployments/${id}/events`);
      return response.events;
    },
  },
};

export { APIError };

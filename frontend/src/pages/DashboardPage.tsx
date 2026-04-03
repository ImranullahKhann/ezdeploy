import { useState, useEffect, FormEvent } from 'react';
import { api, Project } from '../api/client';
import { Navigation } from '../components/Navigation';
import { APIError } from '../api/client';

export function DashboardPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');
  const [formData, setFormData] = useState({
    name: '',
    gitRepoUrl: '',
    branch: 'main',
    workloadType: 'web_service' as const,
  });

  useEffect(() => {
    loadProjects();
  }, []);

  const loadProjects = async () => {
    try {
      const data = await api.projects.list();
      setProjects(data);
      setError('');
    } catch (err) {
      setError('Failed to load projects');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProject = async (e: FormEvent) => {
    e.preventDefault();
    setCreateError('');

    if (!formData.name || !formData.gitRepoUrl || !formData.branch) {
      setCreateError('All fields are required');
      return;
    }

    setCreating(true);
    try {
      const newProject = await api.projects.create(
        formData.name,
        formData.gitRepoUrl,
        formData.branch,
        formData.workloadType
      );
      setProjects([...projects, newProject]);
      setShowCreateForm(false);
      setFormData({ name: '', gitRepoUrl: '', branch: 'main', workloadType: 'web_service' });
    } catch (err) {
      if (err instanceof APIError) {
        setCreateError(err.message);
      } else {
        setCreateError('Failed to create project');
      }
    } finally {
      setCreating(false);
    }
  };

  return (
    <>
      <Navigation />
      <main className="dashboard">
        <div className="dashboard-header">
          <h1>Projects</h1>
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="btn-primary"
          >
            {showCreateForm ? 'Cancel' : 'New Project'}
          </button>
        </div>

        {showCreateForm && (
          <div className="create-project-form">
            <h2>Create New Project</h2>
            {createError && (
              <div className="error-message" role="alert">
                {createError}
              </div>
            )}
            <form onSubmit={handleCreateProject}>
              <div className="form-group">
                <label htmlFor="name">Project Name</label>
                <input
                  id="name"
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  required
                  disabled={creating}
                  placeholder="my-app"
                />
              </div>

              <div className="form-group">
                <label htmlFor="gitRepoUrl">Git Repository URL</label>
                <input
                  id="gitRepoUrl"
                  type="url"
                  value={formData.gitRepoUrl}
                  onChange={(e) =>
                    setFormData({ ...formData, gitRepoUrl: e.target.value })
                  }
                  required
                  disabled={creating}
                  placeholder="https://github.com/user/repo.git"
                />
              </div>

              <div className="form-group">
                <label htmlFor="branch">Branch</label>
                <input
                  id="branch"
                  type="text"
                  value={formData.branch}
                  onChange={(e) =>
                    setFormData({ ...formData, branch: e.target.value })
                  }
                  required
                  disabled={creating}
                  placeholder="main"
                />
              </div>

              <div className="form-group">
                <label htmlFor="workloadType">Workload Type</label>
                <select
                  id="workloadType"
                  value={formData.workloadType}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      workloadType: e.target.value as any,
                    })
                  }
                  required
                  disabled={creating}
                >
                  <option value="web_service">Web Service</option>
                  <option value="static_site">Static Site</option>
                </select>
              </div>

              <button type="submit" disabled={creating} className="btn-primary">
                {creating ? 'Creating...' : 'Create Project'}
              </button>
            </form>
          </div>
        )}

        {loading ? (
          <div className="loading-container">
            <p>Loading projects...</p>
          </div>
        ) : error ? (
          <div className="error-message">{error}</div>
        ) : projects.length === 0 ? (
          <div className="empty-state">
            <p>No projects yet</p>
            <p className="empty-state-hint">
              Create your first project to get started
            </p>
          </div>
        ) : (
          <div className="projects-list">
            {projects.map((project) => (
              <div key={project.id} className="project-card">
                <div className="project-card-header">
                  <h3>{project.name}</h3>
                  <span className={`workload-badge ${project.workload_type}`}>
                    {project.workload_type === 'web_service'
                      ? 'Web Service'
                      : 'Static Site'}
                  </span>
                </div>
                <div className="project-details">
                  <p>
                    <strong>Repository:</strong> {project.git_repo_url}
                  </p>
                  <p>
                    <strong>Branch:</strong> {project.branch}
                  </p>
                  <p className="project-date">
                    Created {new Date(project.created_at).toLocaleDateString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}

      </main>
    </>
  );
}

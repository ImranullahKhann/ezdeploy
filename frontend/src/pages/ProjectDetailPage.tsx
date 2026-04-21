import { useState, useEffect, FormEvent } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { api, Project, ProjectConfig, Deployment, DeploymentEvent, APIError, API_BASE } from '../api/client';
import { Navigation } from '../components/Navigation';

export function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  
  const [project, setProject] = useState<Project | null>(null);
  const [config, setConfig] = useState<ProjectConfig | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [events, setEvents] = useState<DeploymentEvent[]>([]);
  
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [error, setError] = useState('');
  
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>([]);

  useEffect(() => {
    if (id) {
      loadData();
    }
  }, [id]);

  // Poll for deployment status if active
  useEffect(() => {
    let interval: number | undefined;
    const latest = deployments[0];
    if (latest && (latest.status === 'queued' || latest.status === 'building' || latest.status === 'deploying')) {
      interval = window.setInterval(loadDeployments, 3000);
    }
    return () => clearInterval(interval);
  }, [deployments[0]?.status]);

  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const [projData, confData, deploysData] = await Promise.all([
        api.projects.get(id),
        api.projects.getConfig(id),
        api.projects.listDeployments(id)
      ]);
      setProject(projData);
      
      // Set default build_method if not present
      if (!confData.build_method) {
        confData.build_method = 'dockerfile';
      }
      setConfig(confData);
      setDeployments(deploysData);
      
      // Parse env vars
      if (confData.env_vars) {
        const vars = Object.entries(confData.env_vars).map(([key, value]) => ({
          key,
          value: String(value)
        }));
        setEnvVars(vars.length > 0 ? vars : [{ key: '', value: '' }]);
      } else {
        setEnvVars([{ key: '', value: '' }]);
      }

      if (deploysData.length > 0) {
        const eventsData = await api.deployments.listEvents(deploysData[0].id);
        setEvents(eventsData);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load project details');
    } finally {
      setLoading(false);
    }
  };

  const loadDeployments = async () => {
    if (!id) return;
    try {
      const deploysData = await api.projects.listDeployments(id);
      setDeployments(deploysData);
      if (deploysData.length > 0) {
        const eventsData = await api.deployments.listEvents(deploysData[0].id);
        setEvents(eventsData);
      }
    } catch (err) {
      console.error('Failed to reload deployments', err);
    }
  };

  const handleSaveConfig = async (e: FormEvent) => {
    e.preventDefault();
    if (!id || !config || !project) return;
    
    setSaving(true);
    setError('');
    
    try {
      // 1. Update project main fields
      const updatedProject = await api.projects.update(
        id,
        project.name,
        project.git_repo_url,
        project.branch,
        project.workload_type
      );
      setProject(updatedProject);

      // 2. Update config and env vars
      const envObj: Record<string, string> = {};
      envVars.forEach(v => {
        if (v.key.trim()) envObj[v.key.trim()] = v.value;
      });

      const updatedConfig = await api.projects.updateConfig(id, {
        ...config,
        env_vars: envObj
      });
      setConfig(updatedConfig);

      alert('Configuration saved successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save configuration');
    } finally {
      setSaving(false);
    }
  };

  const handleDeploy = async () => {
    if (!id) return;
    setDeploying(true);
    try {
      const dep = await api.projects.deploy(id);
      setDeployments([dep, ...deployments]);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to trigger deployment');
    } finally {
      setDeploying(false);
    }
  };

  const addEnvVar = () => setEnvVars([...envVars, { key: '', value: '' }]);
  const removeEnvVar = (index: number) => {
    const newVars = [...envVars];
    newVars.splice(index, 1);
    setEnvVars(newVars.length > 0 ? newVars : [{ key: '', value: '' }]);
  };
  const updateEnvVar = (index: number, field: 'key' | 'value', val: string) => {
    const newVars = [...envVars];
    newVars[index][field] = val;
    setEnvVars(newVars);
  };

  if (loading) return (
    <>
      <Navigation />
      <div className="loading-container"><p>Loading project details...</p></div>
    </>
  );

  if (error && !project) return (
    <>
      <Navigation />
      <div className="error-container">
        <p>{error}</p>
        <button onClick={() => navigate('/dashboard')} className="btn-secondary">Back to Dashboard</button>
      </div>
    </>
  );

  if (!project || !config) return null;

  const latestDeployment = deployments[0];

  return (
    <>
      <Navigation />
      <main className="project-detail">
        <div className="detail-header">
          <div className="header-left">
            <Link to="/dashboard" className="back-link">← Back</Link>
            <h1>{project.name}</h1>
            <span className={`workload-badge ${project.workload_type}`}>
              {project.workload_type === 'web_service' ? 'Web Service' : 'Static Site'}
            </span>
          </div>
          <div className="header-right">
            <button 
              onClick={handleDeploy} 
              disabled={deploying || (latestDeployment?.status === 'queued' || latestDeployment?.status === 'building' || latestDeployment?.status === 'deploying')}
              className="btn-primary"
            >
              {deploying ? 'Triggering...' : 'Deploy Now'}
            </button>
          </div>
        </div>

        <div className="detail-grid">
          <section className="config-section card">
            <h2>Configuration</h2>
            <form onSubmit={handleSaveConfig}>
              <div className="form-group">
                <label>Project Name</label>
                <input 
                  type="text" 
                  value={project.name} 
                  onChange={(e) => setProject({...project, name: e.target.value})}
                  required
                />
              </div>
              <div className="form-group">
                <label>Repository URL</label>
                <input type="text" value={project.git_repo_url} disabled />
              </div>
              <div className="form-group">
                <label>Target Branch</label>
                <input 
                  type="text" 
                  value={project.branch} 
                  onChange={(e) => setProject({...project, branch: e.target.value})}
                  required
                />
              </div>

              {project.workload_type === 'web_service' ? (
                <>
                  <div className="form-group">
                    <label>Build Method</label>
                    <div className="radio-group">
                      <label className="radio-option">
                        <input 
                          type="radio" 
                          name="build_method" 
                          value="dockerfile"
                          checked={config.build_method === 'dockerfile'}
                          onChange={(e) => setConfig({...config, build_method: e.target.value})}
                        />
                        <span>Dockerfile</span>
                        <small>Build using an existing Dockerfile in your repository</small>
                      </label>
                      <label className="radio-option">
                        <input 
                          type="radio" 
                          name="build_method" 
                          value="buildpack"
                          checked={config.build_method === 'buildpack'}
                          onChange={(e) => setConfig({...config, build_method: e.target.value})}
                        />
                        <span>Build Commands</span>
                        <small>Automatically generate a Dockerfile from build and start commands</small>
                      </label>
                    </div>
                  </div>

                  {config.build_method === 'dockerfile' ? (
                    <>
                      <div className="form-group">
                        <label>Dockerfile Path</label>
                        <input 
                          type="text" 
                          value={config.dockerfile_path || ''} 
                          onChange={(e) => setConfig({...config, dockerfile_path: e.target.value})}
                          placeholder="Dockerfile"
                        />
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="form-group">
                        <label>Install Command</label>
                        <input 
                          type="text" 
                          value={config.install_cmd || ''} 
                          onChange={(e) => setConfig({...config, install_cmd: e.target.value})}
                          placeholder="npm install"
                        />
                        <small className="field-hint">Command to install dependencies (optional)</small>
                      </div>
                      <div className="form-group">
                        <label>Build Command</label>
                        <input 
                          type="text" 
                          value={config.build_cmd || ''} 
                          onChange={(e) => setConfig({...config, build_cmd: e.target.value})}
                          placeholder="npm run build"
                        />
                        <small className="field-hint">Command to build your application (optional)</small>
                      </div>
                      <div className="form-group">
                        <label>Start Command</label>
                        <input 
                          type="text" 
                          value={config.start_cmd || ''} 
                          onChange={(e) => setConfig({...config, start_cmd: e.target.value})}
                          placeholder="npm start"
                          required
                        />
                        <small className="field-hint">Command to start your application</small>
                      </div>
                    </>
                  )}

                  <div className="form-group">
                    <label>Container Port</label>
                    <input 
                      type="number" 
                      value={config.port || ''} 
                      onChange={(e) => setConfig({...config, port: parseInt(e.target.value) || 0})}
                      placeholder="8080"
                    />
                  </div>
                  <div className="form-group">
                    <label>Healthcheck Path</label>
                    <input 
                      type="text" 
                      value={config.healthcheck_path || ''} 
                      onChange={(e) => setConfig({...config, healthcheck_path: e.target.value})}
                      placeholder="/"
                    />
                  </div>
                </>
              ) : (
                <>
                  <div className="form-group">
                    <label>Install Command</label>
                    <input 
                      type="text" 
                      value={config.install_cmd || ''} 
                      onChange={(e) => setConfig({...config, install_cmd: e.target.value})}
                      placeholder="npm install"
                    />
                  </div>
                  <div className="form-group">
                    <label>Build Command</label>
                    <input 
                      type="text" 
                      value={config.build_cmd || ''} 
                      onChange={(e) => setConfig({...config, build_cmd: e.target.value})}
                      placeholder="npm run build"
                    />
                  </div>
                  <div className="form-group">
                    <label>Output Directory</label>
                    <input 
                      type="text" 
                      value={config.output_dir || ''} 
                      onChange={(e) => setConfig({...config, output_dir: e.target.value})}
                      placeholder="dist"
                    />
                  </div>
                </>
              )}

              <div className="env-vars-section">
                <h3>Environment Variables (Secrets)</h3>
                <p className="section-hint">These variables will be injected into your build and runtime environment.</p>
                {envVars.map((v, i) => (
                  <div key={i} className="env-var-row">
                    <input 
                      type="text" 
                      placeholder="KEY" 
                      value={v.key} 
                      onChange={(e) => updateEnvVar(i, 'key', e.target.value.toUpperCase())}
                    />
                    <input 
                      type="password" 
                      placeholder="VALUE" 
                      value={v.value} 
                      onChange={(e) => updateEnvVar(i, 'value', e.target.value)}
                    />
                    <button type="button" onClick={() => removeEnvVar(i)} className="btn-icon">×</button>
                  </div>
                ))}
                <button type="button" onClick={addEnvVar} className="btn-secondary btn-small">Add Variable</button>
              </div>

              <div className="form-actions">
                <button type="submit" disabled={saving} className="btn-primary">
                  {saving ? 'Saving...' : 'Save Configuration'}
                </button>
              </div>
            </form>
          </section>

          <div className="status-column">
            <section className="current-status card">
              <h2>Current Status</h2>
              {latestDeployment ? (
                <div className="status-display">
                  <div className={`status-banner ${latestDeployment.status}`}>
                    {latestDeployment.status.toUpperCase()}
                  </div>
                  {latestDeployment.status === 'running' && latestDeployment.public_url && (
                    <div className="public-url-box">
                      <label>Public URL</label>
                      {(() => {
                        const baseUrl = API_BASE.endsWith('/') ? API_BASE.slice(0, -1) : API_BASE;
                        const publicPath = latestDeployment.public_url.startsWith('/') 
                          ? latestDeployment.public_url 
                          : `/${latestDeployment.public_url}`;
                        const fullUrl = latestDeployment.public_url.startsWith('http') 
                          ? latestDeployment.public_url 
                          : `${baseUrl}${publicPath}`;
                        return (
                          <a href={fullUrl} target="_blank" rel="noopener noreferrer">
                            {fullUrl} ↗
                          </a>
                        );
                      })()}
                    </div>
                  )}
                  <div className="deployment-meta">
                    <p><strong>Deployment ID:</strong> {latestDeployment.id}</p>
                    <p><strong>Created:</strong> {new Date(latestDeployment.created_at).toLocaleString()}</p>
                  </div>
                </div>
              ) : (
                <p className="empty-text">No deployments yet. Click "Deploy Now" to start.</p>
              )}
            </section>

            <section className="events-section card">
              <h2>Recent Events</h2>
              <div className="event-list">
                {events.length > 0 ? events.slice().reverse().map(ev => (
                  <div key={ev.id} className="event-item">
                    <span className="event-time">{new Date(ev.timestamp).toLocaleTimeString()}</span>
                    <span className="event-msg">{ev.message}</span>
                  </div>
                )) : (
                  <p className="empty-text">No events to show.</p>
                )}
              </div>
            </section>
          </div>
        </div>
      </main>
    </>
  );
}

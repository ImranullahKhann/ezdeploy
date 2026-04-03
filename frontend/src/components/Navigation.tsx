import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export function Navigation() {
  const { user, logout } = useAuth();

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  if (!user) {
    return null;
  }

  return (
    <nav className="navigation">
      <div className="nav-content">
        <Link to="/dashboard" className="nav-brand">
          ezdeploy
        </Link>
        <div className="nav-links">
          <Link to="/dashboard">Projects</Link>
          <div className="nav-user">
            <span>{user.email}</span>
            <button onClick={handleLogout} className="btn-link">
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}

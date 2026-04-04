# Frontend Overview

The `ezdeploy` frontend is a modern Single Page Application (SPA) built with React and TypeScript.

## Tech Stack

- **React**: Component-based UI library.
- **TypeScript**: Static typing for enhanced developer experience.
- **Vite**: Fast build tool and development server.
- **React Router**: Client-side routing for multi-page feel.
- **Vanilla CSS**: Stylized components and layouts.
- **Lucide React**: Icon library for a polished look.

## Project Structure

- `src/api/`: Contains the centralized `api` object for all backend communications. It uses the `fetch` API and handles session credentials.
- `src/components/`: Reusable UI components like `Navigation`, `ProtectedRoute`, and layout elements.
- `src/context/`: `AuthContext` provides global authentication state and helper functions like `login`, `signup`, and `logout`.
- `src/pages/`: Main views of the application:
    - **DashboardPage**: Lists all user projects and provides an interface to create new ones.
    - **ProjectDetailPage**: Detailed view of a project, showing deployment history, project configurations, and the ability to trigger new deployments.
    - **LoginPage** & **SignupPage**: User authentication pages.
- `src/App.tsx`: Main application entry point, defining routes and providing the authentication context.

## Authentication & Authorization

- **`AuthContext`**: Manages the `user` state. It attempts to fetch the current user profile on initial load using `api.auth.me`.
- **`ProtectedRoute`**: A wrapper component that redirects unauthenticated users to the `/login` page if they try to access protected routes like `/dashboard`.
- **Credentials**: All API requests are made with `credentials: 'include'` to ensure session cookies are sent to the backend.

## API Interactions

The frontend interacts with the backend through the `api` object in `src/api/client.ts`. It follows the RESTful patterns defined in the [API Reference](./api-reference.md).

Example:
```typescript
import { api } from '../api/client';

const projects = await api.projects.list();
```

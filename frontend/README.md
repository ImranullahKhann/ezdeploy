# Frontend

React UI for ezdeploy.

## Phase 4 Implementation

This frontend application implements Phase 4 of the BUILD_PLAN:

### Card 4.1 — Scaffold React application ✓
- Vite + React 19 + TypeScript setup
- React Router v7 for routing
- API client layer (`src/api/client.ts`) with typed endpoints
- Global layout and navigation components

### Card 4.2 — Implement auth screens ✓
- Sign up page (`/signup`)
- Sign in page (`/login`)
- Form validation with real-time feedback
- Error display for authentication failures
- Full integration with backend auth API

### Card 4.3 — Implement authenticated app shell ✓
- Navigation component with user account display
- Auth context provider for global authentication state
- Protected route guards that redirect unauthenticated users
- Loading states during auth checks
- Logout functionality

### Card 4.4 — Implement project list and create project UI ✓
- Dashboard page showing all user projects
- Create project form with validation
- Project list with cards displaying project details
- Empty state for users with no projects
- Real-time updates after creating projects
- Full integration with backend project API

## Structure

```
src/
├── api/
│   └── client.ts          # API client with typed methods
├── components/
│   ├── Navigation.tsx     # Top navigation bar
│   └── ProtectedRoute.tsx # Route guard component
├── context/
│   └── AuthContext.tsx    # Auth state management
├── pages/
│   ├── DashboardPage.tsx  # Project list and creation
│   ├── LoginPage.tsx      # Sign in form
│   └── SignupPage.tsx     # Sign up form
├── App.tsx                # Main app with routing
├── main.tsx               # Entry point
├── styles.css             # Global styles
└── vite-env.d.ts          # TypeScript definitions
```

## Development

```bash
npm run dev       # Start dev server on port 5173
npm run build     # Build for production
npm run preview   # Preview production build
npm run typecheck # Run TypeScript type checking
npm test          # Run typecheck and build
```

## API Integration

The frontend communicates with the backend API at `http://localhost:8080` by default. This can be configured via the `VITE_API_BASE_URL` or `VITE_API_BASE` environment variable.

All API calls include credentials (cookies) for session management.

## Routes

- `/` - Redirects to `/dashboard`
- `/login` - Sign in page (public)
- `/signup` - Sign up page (public)
- `/dashboard` - Project list and management (protected)


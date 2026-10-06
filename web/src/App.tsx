import { Navigate, Route, Routes } from 'react-router-dom'
import StudentLogin from './pages/StudentLogin'
import StudentLobby from './pages/StudentLobby'
import StudentWorkspace from './pages/StudentWorkspace'
import AdminLogin from './pages/AdminLogin'
import AdminLayout from './pages/AdminLayout'
import AdminDashboard from './pages/AdminDashboard'
import AdminQuestions from './pages/AdminQuestions'
import AdminConfig from './pages/AdminConfig'
import AdminLeaderboard from './pages/AdminLeaderboard'
import LeaderboardPresent from './pages/LeaderboardPresent'
import { getAdminToken, getStudentToken } from './api'

function RequireStudent({ children }: { children: React.ReactNode }) {
  if (!getStudentToken()) return <Navigate to="/" replace />
  return <>{children}</>
}

function RequireAdmin({ children }: { children: React.ReactNode }) {
  if (!getAdminToken()) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<StudentLogin />} />
      <Route
        path="/lobby"
        element={
          <RequireStudent>
            <StudentLobby />
          </RequireStudent>
        }
      />
      <Route
        path="/contest"
        element={
          <RequireStudent>
            <StudentWorkspace />
          </RequireStudent>
        }
      />
      <Route path="/admin/login" element={<AdminLogin />} />
      <Route
        path="/admin/present"
        element={
          <RequireAdmin>
            <LeaderboardPresent />
          </RequireAdmin>
        }
      />
      <Route
        path="/admin"
        element={
          <RequireAdmin>
            <AdminLayout />
          </RequireAdmin>
        }
      >
        <Route index element={<AdminDashboard />} />
        <Route path="questions" element={<AdminQuestions />} />
        <Route path="config" element={<AdminConfig />} />
        <Route path="leaderboard" element={<AdminLeaderboard />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

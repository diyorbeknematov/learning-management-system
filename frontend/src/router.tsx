import { createBrowserRouter } from 'react-router'
import { GuestOnly, RequireAuth } from '@/auth/guards'
import { AppLayout } from '@/components/layout/AppLayout'

// Each page is loaded when it is opened, so the first download stays small.
const page = (load: () => Promise<{ default: React.ComponentType }>) => async () => ({
  Component: (await load()).default,
})

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, lazy: page(() => import('@/pages/HomePage')) },
      { path: 'courses/:courseId', lazy: page(() => import('@/pages/CoursePage')) },
      { path: 'verify', lazy: page(() => import('@/pages/VerifyPage')) },
      { path: 'verify/:uniqueId', lazy: page(() => import('@/pages/VerifyPage')) },

      // only for visitors
      {
        element: <GuestOnly />,
        children: [
          { path: 'login', lazy: page(() => import('@/pages/LoginPage')) },
          { path: 'register', lazy: page(() => import('@/pages/RegisterPage')) },
        ],
      },

      // the link of the email opens this page, logged in or not
      { path: 'forgot-password', lazy: page(() => import('@/pages/ForgotPasswordPage')) },
      { path: 'reset-password', lazy: page(() => import('@/pages/ResetPasswordPage')) },

      // only for logged in users
      {
        element: <RequireAuth />,
        children: [
          { path: 'dashboard', lazy: page(() => import('@/pages/DashboardPage')) },
          { path: 'profile', lazy: page(() => import('@/pages/ProfilePage')) },
          { path: 'quizzes/:quizId', lazy: page(() => import('@/pages/QuizPage')) },
        ],
      },

      // only for students
      {
        element: <RequireAuth roles={['Student']} />,
        children: [
          { path: 'my-courses', lazy: page(() => import('@/pages/MyCoursesPage')) },
          { path: 'learn/:courseId', lazy: page(() => import('@/pages/LearnPage')) },
          { path: 'learn/:courseId/lessons/:lessonId', lazy: page(() => import('@/pages/LearnPage')) },
          { path: 'certificates', lazy: page(() => import('@/pages/CertificatesPage')) },
        ],
      },

      // for instructors (a SuperAdmin can do the same)
      {
        element: <RequireAuth roles={['Instructor']} />,
        children: [
          { path: 'teach/courses', lazy: page(() => import('@/pages/teach/TeachCoursesPage')) },
          { path: 'teach/courses/new', lazy: page(() => import('@/pages/teach/NewCoursePage')) },
          { path: 'teach/courses/:courseId', lazy: page(() => import('@/pages/teach/CourseEditorPage')) },
        ],
      },

      // only for the SuperAdmin
      {
        element: <RequireAuth roles={['SuperAdmin']} />,
        children: [
          { path: 'admin/users', lazy: page(() => import('@/pages/admin/UsersPage')) },
          { path: 'admin/categories', lazy: page(() => import('@/pages/admin/CategoriesPage')) },
          { path: 'admin/payments', lazy: page(() => import('@/pages/admin/PaymentsPage')) },
          { path: 'admin/finance', lazy: page(() => import('@/pages/admin/FinancePage')) },
          { path: 'admin/reports', lazy: page(() => import('@/pages/admin/ReportsPage')) },
        ],
      },

      { path: '*', lazy: page(() => import('@/pages/NotFoundPage')) },
    ],
  },
])

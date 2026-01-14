import { createBrowserRouter, RouterProvider } from "react-router-dom";
import './index.css';
import Home from './pages/Home';
import Workspace from "./pages/Workspace";
import { SocketProvider } from './utils/Socket';

const router = createBrowserRouter([
  {
    path: "/",
    element: <Home />,
  },
  {
    path: "/workspace/:workspaceId",
    element: <Workspace/>,
  },
]);

function App() {
  return (
    <SocketProvider>
      <RouterProvider router={router} />
    </SocketProvider>
  );
}

export default App;
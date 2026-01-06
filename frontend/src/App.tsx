import {
  createBrowserRouter,
  RouterProvider,
} from "react-router-dom";
import './App.css'
import Home from './pages/Home';
import { Socket } from './utils/Socket';

const router = createBrowserRouter([
  {
    path: "/",
    element: <Home />,
  },
]);

function App() {

  return (
    <>
      <Socket>
        <RouterProvider router={router} />    
      </Socket>
    </>
  )
}

export default App

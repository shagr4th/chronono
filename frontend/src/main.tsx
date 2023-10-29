import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { MantineColorsTuple, MantineProvider, createTheme } from '@mantine/core'


const mainColor: MantineColorsTuple = [
  "#f3f3fe",
  "#e4e6ed",
  "#c8cad3",
  "#a9adb9",
  "#9093a4",
  "#808496",
  "#767c91",
  "#656a7e",
  "#585e72",
  "#4a5167"
]

const theme = createTheme({
  colors: {
    mainColor: mainColor
  },
  fontFamily: 'Intel Mono',
  primaryColor: 'mainColor',
})

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <MantineProvider defaultColorScheme="auto" theme={theme}>
      <App />
    </MantineProvider>
  </React.StrictMode>,
)

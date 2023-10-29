import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { MantineColorsTuple, MantineProvider, createTheme, rem } from '@mantine/core'


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
  fontFamily: 'Martian Mono',
  fontSizes: {
    xs: rem(10),
    sm: rem(11),
    md: rem(14),
    lg: rem(16),
    xl: rem(20),
  },
  primaryColor: 'mainColor',
})

ReactDOM.createRoot(document.body).render(
  <React.StrictMode>
    <MantineProvider defaultColorScheme="auto" theme={theme}>
      <App />
    </MantineProvider>
  </React.StrictMode>,
)

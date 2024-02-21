import { useEffect, useRef, useState } from 'react'
import './App.css'
import { ActionIcon, Button, Flex, Group, Notification, NumberInput, RingProgress, Text, TextInput, Textarea, Tooltip, useComputedColorScheme, useMantineColorScheme } from '@mantine/core';
import { IconPlayerPlayFilled, IconPlayerStopFilled, IconRewindForward60, IconRewindForward10, IconArrowForwardUp,
  IconRewindBackward60, IconRewindBackward10, IconArrowBackUp, IconClockHour12, IconSun, IconMoonStars, IconSend } from '@tabler/icons-react'
import { useDisclosure, useResizeObserver } from '@mantine/hooks';


const ColorSchemeButton = () => {
  const { setColorScheme, clearColorScheme } = useMantineColorScheme()
  const computedColorScheme = useComputedColorScheme('dark', { getInitialValueInEffect: true })

  return <ActionIcon
      style={{
          marginLeft: 'auto'
      }}
      variant="outline"
      color={computedColorScheme == 'dark' ? 'yellow' : 'blue'}
      onClick={() => setColorScheme(computedColorScheme === 'dark' ? 'light' : 'dark')}
      title="Toggle color scheme"
  >{computedColorScheme == 'dark' ? <IconSun size="1.0rem" /> : <IconMoonStars size="1.0rem" />}
  </ActionIcon>
}

function App() {
  const [hours, setHours] = useState(0);
  const [minutes, setMinutes] = useState(0)
  const [seconds, setSeconds] = useState(0)
  const [oscclients, setOscClients] = useState<string | null>(localStorage.getItem("oscclients"))
  const [logs, setLogs] = useState<string[]>([])
  const [info, setInfo] = useState("")
  const [infoInError, setInfoInError] = useState(false)
  const [time, setTime] = useState(0)
  const [clockRef, { width: clockWidth }] = useResizeObserver<HTMLDivElement>();

  const [timeEntryOpened, timeEntryHandlers] = useDisclosure(false);
  const setTimeFunction = (newTime: number) => {
    if (newTime < 0) {
      newTime = 0;
    }
    fetch("/config?time=" + 1000 * newTime);
  }

  useEffect(() => {
    if (oscclients) {
      localStorage.setItem("oscclients", oscclients);
    } else {
      localStorage.removeItem("oscclients")
    }
  }, [oscclients])

  useEffect(() => {
    var h = Math.floor(time / 3600);
    setHours(h);
    setMinutes(Math.floor((time - h * 3600) / 60));
    setSeconds(Math.floor(time % 60));
  }, [time]);

  useEffect(() => {

    var sse = new EventSource("/sse");
    sse.onopen = function (evt) {
      if (oscclients) {
        fetch("/config?clients=" + oscclients);
      }
    }
    sse.onmessage = function (evt) {
      if (evt.data && evt.data.lastIndexOf('time=', 0) === 0) {
        var timeInMs = parseInt(evt.data.substring(5));
        if (timeInMs >= 0)
          setTime(timeInMs / 1000);
        else
          setTime(0)
      } else if (evt.data && evt.data.lastIndexOf('http=', 0) === 0) {
        setInfo(evt.data.substring(5));
        setInfoInError(false)
      } else {
        setLogs(logs => logs.concat('[' + new Date().toTimeString().substring(0, 8) + '] ' + evt.data));
      }
    }
    sse.onerror = function (evt) {
      setInfo("Connection error")
      setInfoInError(true)
    }

    return () => {
      sse.close();
    };
  }, [])

  return (
    <Flex direction="column" w="100%" h="100vh">
      <Group justify="end" mt={10} mr={10}>
        <ColorSchemeButton/>
      </Group>
      <Group justify="center" align="center" ref={clockRef} onClick={(evt) => {
        if (!(evt.target instanceof HTMLInputElement)) {
          timeEntryHandlers.toggle()
        }
      }} style={{
        height: ((Math.min(Math.max(clockWidth, 128), 1024) - 20) / 2) + "px"
      }}>

        {!timeEntryOpened && <RingProgress size={(Math.min(Math.max(clockWidth, 128), 1024) - 20) / 2} thickness={10} roundCaps m={0}
            sections={[{ value: 100 * minutes / 60, color: 'blue' }]}
            label={
              <Text fz={Math.min(Math.max(clockWidth, 128), 1024) / 5} fw={700} ta="center" size="xl">
                {('0' + Math.floor(minutes)).slice(-2)}
              </Text>
            }
          />}

        {timeEntryOpened && <NumberInput style={{
          width: ((Math.min(Math.max(clockWidth, 128), 1024) - 40) / 2) + "px",
          margin: "5px"
        }}
          label="Minutes" autoFocus={true} value={minutes} onChange={(evt) => {
            const m = typeof evt == "number" ? evt as number : 0
            setMinutes(m)
            setTimeFunction(m * 60 + seconds)
          }} onKeyDown={(evt) => {
            if (evt.key === 'Enter') {
              timeEntryHandlers.close()
            }
          }
        }></NumberInput>}

        {!timeEntryOpened && <RingProgress size={(Math.min(Math.max(clockWidth, 128), 1024) - 20) / 2} thickness={10} roundCaps m={0}
            sections={[{ value: 100 * seconds / 60, color: 'blue' }]}
            label={
              <Text fz={Math.min(Math.max(clockWidth, 128), 1024) / 5}  fw={700} ta="center" size="xl">
                {('0' + Math.floor(seconds)).slice(-2)}
              </Text>
            }
          />}

        {timeEntryOpened && <NumberInput style={{
          width: ((Math.min(Math.max(clockWidth, 128), 1024) - 40) / 2) + "px",
          margin: "5px"
        }}
          label="Seconds" value={seconds} onChange={(evt) => {
            const s = typeof evt == "number" ? evt as number : 0
            setSeconds(s)
            setTimeFunction(minutes * 60 + s)
          }} onKeyDown={(evt) => {
            if (evt.key === 'Enter') {
              timeEntryHandlers.close()
            }
          }
        }></NumberInput>}

      </Group>

      <Group justify="space-around" mt={10}>

        <ActionIcon variant="subtle" size="xl" onClick={() => setTimeFunction(time - 60)}>
          <IconRewindBackward60 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindBackward60>
        </ActionIcon>
        <ActionIcon variant="subtle" size="xl" onClick={() => setTimeFunction(time - 10)}>
          <IconRewindBackward10 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindBackward10>
        </ActionIcon>
        <ActionIcon variant="subtle"  size="xl" onClick={() => setTimeFunction(time - 1)}>
          <IconArrowBackUp style={{ width: '70%', height: '70%' }} stroke={1.5}></IconArrowBackUp>
        </ActionIcon>
        <ActionIcon variant="subtle" size="xl"  onClick={() => setTimeFunction(time + 1)}>
          <IconArrowForwardUp style={{ width: '70%', height: '70%' }} stroke={1.5}></IconArrowForwardUp>
        </ActionIcon>
        <ActionIcon variant="subtle" size="xl"  onClick={() => setTimeFunction(time + 10)} >
          <IconRewindForward10 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindForward10>
        </ActionIcon>
        <ActionIcon variant="subtle" size="xl"  onClick={() => setTimeFunction(time + 60)} >
          <IconRewindForward60 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindForward60>
        </ActionIcon>

      </Group>

      <Group justify="center" mt={10}>

        <Button variant="gradient" size="lg" onClick={() => fetch("/start")} leftSection={
          <IconPlayerPlayFilled size={24}></IconPlayerPlayFilled>
        }>Start</Button>
        <Button variant="gradient" size="lg" onClick={() => fetch("/stop")} leftSection={
          <IconPlayerStopFilled size={24}></IconPlayerStopFilled>
        }>Stop</Button>
        <Button variant="gradient" size="lg" onClick={() => fetch("/reset")} leftSection={
          <IconClockHour12 size={24}></IconClockHour12>
        }>Reset</Button>

      </Group>

      {info && <Notification color={infoInError ? 'red' : 'blue'} title="Information" withCloseButton={false} mt={10}>
        { info }
      </Notification>}

      <TextInput mx={5} mt={10} placeholder='Press enter to save'
        label="Client(s) OSC" value={oscclients || ""} onChange={(evt) =>
          setOscClients(evt.currentTarget.value)} onKeyDown={(evt) =>
            evt.key === 'Enter' && fetch("/config?clients=" + oscclients)
        }
      />

      <Textarea label="Logs" mx={5} mt={10} value={logs.join('\n')} style={{
        flexGrow: 1,
        display: "flex",
        flexDirection: "column"
      }} styles={{
        wrapper: {
          flexGrow: 1,
          display: "flex",
          flexDirection: "column"
        },
        input: {
          flexGrow: 1
        }
      }}></Textarea>

      <Button m={5} variant="subtle" onClick={() => setLogs([])}>Clear logs</Button>
    </Flex>
  )
}

export default App

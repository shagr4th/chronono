import { useEffect, useRef, useState } from 'react'
import './App.css'
import { ActionIcon, Button, Flex, Group, Notification, RingProgress, Text, TextInput, Textarea, useComputedColorScheme, useMantineColorScheme } from '@mantine/core';
import { IconPlayerPlayFilled, IconPlayerStopFilled, IconRewindForward60, IconRewindForward10, IconArrowForwardUp,
  IconRewindBackward60, IconRewindBackward10, IconArrowBackUp, IconClockHour12, IconSun, IconMoonStars } from '@tabler/icons-react'
import { useResizeObserver } from '@mantine/hooks';


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
  const logsRef = useRef<string[]>();

  useEffect(() => {
    logsRef.current = logs;
  }, [logs]);

  const addTimeFunction = (delta: number) => {
    var newTime = time + delta;
    if (newTime < 0) {
      newTime = 0;
    }
    fetch("/config?time=" + 1000 * newTime);
  }

  useEffect(() => {
    if (oscclients) {
      localStorage.setItem("oscclients", oscclients);
      fetch("/config?clients=" + oscclients);
    } else {
      localStorage.removeItem("oscclients")
    }
  }, [oscclients])

  useEffect(() => {
    var h = Math.floor(time / 3600);
    setHours(h);
    setMinutes(Math.floor((time - h * 3600) / 60));
    setSeconds(time % 60);
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
        setLogs((logsRef.current ?? []).concat('[' + new Date().toTimeString().substring(0, 8) + '] ' + evt.data))
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
    <Flex direction="column" w="100%" h="100%">
      <Group justify="end" mt={10} mr={10}>
        <ColorSchemeButton/>
      </Group>
      <Group justify="center" ref={clockRef}>

      { clockWidth > 100 && <RingProgress size={(clockWidth - 20) / 2} thickness={10} roundCaps m={0}
          sections={[{ value: 100 * minutes / 60, color: 'blue' }]}
          label={
            <Text fz={clockWidth / 5} fw={700} ta="center" size="xl">
              {('0' + Math.floor(minutes)).slice(-2)}
            </Text>
          }
        />}

      { clockWidth > 100 && <RingProgress size={(clockWidth - 20) / 2} thickness={10} roundCaps m={0}
          sections={[{ value: 100 * seconds / 60, color: 'blue' }]}
          label={
            <Text fz={clockWidth / 5}  fw={700} ta="center" size="xl">
              {('0' + Math.floor(seconds)).slice(-2)}
            </Text>
          }
        />}

      </Group>


      {/*<Group justify="center" mt={10}>
        <TextInput
          label="Client(s) OSC" value={oscclients || ""} onChange={(evt) =>
            setOscClients(evt.target.value)
          }
        />
        </Group>*/}

      <Group justify="center" mt={10}>

        <Button variant="filled" size="lg" onClick={() => fetch("/start")} leftSection={
          <IconPlayerPlayFilled size={24}></IconPlayerPlayFilled>
        }>Start</Button>
        <Button variant="filled" size="lg" onClick={() => fetch("/stop")} leftSection={
          <IconPlayerStopFilled size={24}></IconPlayerStopFilled>
        }>Stop</Button>
        <Button variant="filled" size="lg" onClick={() => fetch("/reset")} leftSection={
          <IconClockHour12 size={24}></IconClockHour12>
        }>Reset</Button>

      </Group>

      <Group justify="center" mt={10}>

        <ActionIcon variant="filled"  size="xl" onClick={() => addTimeFunction(-60)}>
          <IconRewindBackward60 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindBackward60>
        </ActionIcon>
        <ActionIcon variant="filled" size="xl" onClick={() => addTimeFunction(-10)}>
          <IconRewindBackward10 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindBackward10>
        </ActionIcon>
        <ActionIcon variant="filled"  size="xl" onClick={() => addTimeFunction(-1)}>
          <IconArrowBackUp style={{ width: '70%', height: '70%' }} stroke={1.5}></IconArrowBackUp>
        </ActionIcon>
        <ActionIcon variant="filled" size="xl"  onClick={() => addTimeFunction(1)}>
          <IconArrowForwardUp style={{ width: '70%', height: '70%' }} stroke={1.5}></IconArrowForwardUp>
        </ActionIcon>
        <ActionIcon variant="filled" size="xl"  onClick={() => addTimeFunction(10)} >
          <IconRewindForward10 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindForward10>
        </ActionIcon>
        <ActionIcon variant="filled" size="xl"  onClick={() => addTimeFunction(60)} >
          <IconRewindForward60 style={{ width: '70%', height: '70%' }} stroke={1.5}></IconRewindForward60>
        </ActionIcon>

      </Group>

      {info && <Notification color={infoInError ? 'red' : 'blue'} title="Information" withCloseButton={false} mt={10}>
        { info }
      </Notification>}

      <Textarea label="Logs" m={5} value={logs.join('\n')} style={{
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

      <Button m={5} variant="filled" onClick={() => setLogs([])}>Clear logs</Button>
    </Flex>
  )
}

export default App

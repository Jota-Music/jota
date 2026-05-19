import '@/lib/music/views/stores/socket'
import '@/lib/shared/api/socket'
import '@/lib/shared/views/stores/error-listeners'
import '@/lib/shared/views/ui/hooks/tabs'
import { syncAuth } from '@/lib/auth/views/stores/session'
import Router from '@/lib/shared/views/ui/router'
import { render } from 'preact'
import './style.tw.css'

void syncAuth()

render(<Router />, document.getElementById('app')!)


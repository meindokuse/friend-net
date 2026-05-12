import { useState } from 'react'
import { Card } from '../components/Card'
import { Field, SelectField } from '../components/Field'
import { Btn } from '../components/Btn'
import { ResponseBox } from '../components/ResponseBox'
import { usersApi } from '../api'
import type { TokenState } from '../types'

interface Props {
  tokens: TokenState
}

function useCall() {
  const [status, setStatus] = useState<number>()
  const [data, setData] = useState<unknown>()
  const [loading, setLoading] = useState(false)

  async function call(fn: () => Promise<{ status: number; data: unknown; isOk: boolean }>) {
    setLoading(true)
    try {
      const r = await fn()
      setStatus(r.status)
      setData(r.data)
      return r
    } finally {
      setLoading(false)
    }
  }
  return { status, data, loading, call }
}

const PRIVACY_OPTIONS = ['everyone', 'friends', 'nobody']

export function UsersSection({ tokens }: Props) {
  const at = tokens.accessToken

  // Create user
  const create = useCall()
  const [cuUser, setCuUser] = useState('')
  const [cuEmail, setCuEmail] = useState('')
  const [cuPhone, setCuPhone] = useState('')
  const [cuName, setCuName] = useState('')

  // Get me
  const getMe = useCall()

  // Get by ID
  const getById = useCall()
  const [targetId, setTargetId] = useState('')

  // Get by username
  const getByName = useCall()
  const [targetUser, setTargetUser] = useState('')

  // Batch
  const batch = useCall()
  const [batchIds, setBatchIds] = useState('')

  // Update profile
  const updateProf = useCall()
  const [profName, setProfName] = useState('')
  const [profBio, setProfBio] = useState('')
  const [profAvatar, setProfAvatar] = useState('')
  const [profVersion, setProfVersion] = useState('0')

  // Update settings
  const updateSet = useCall()
  const [setMsg, setSetMsg] = useState('everyone')
  const [setLastSeen, setSetLastSeen] = useState('everyone')
  const [setProfile, setSetProfile] = useState('everyone')
  const [setLang, setSetLang] = useState('en')
  const [setTz, setSetTz] = useState('UTC')
  const [setVersion, setSetVersion] = useState('0')

  // Change email
  const chEmail = useCall()
  const [newEmail, setNewEmail] = useState('')
  const [emailVer, setEmailVer] = useState('0')

  // Change phone
  const chPhone = useCall()
  const [newPhone, setNewPhone] = useState('')
  const [phoneVer, setPhoneVer] = useState('0')

  // Delete
  const del = useCall()
  const [delVer, setDelVer] = useState('0')

  // Last seen
  const lastSeen = useCall()

  // Search
  const search = useCall()
  const [searchQ, setSearchQ] = useState('')
  const [searchLimit, setSearchLimit] = useState('20')

  // List
  const list = useCall()
  const [listLimit, setListLimit] = useState('20')

  return (
    <div className="flex flex-col gap-3">
      {/* Create */}
      <Card title="Создать пользователя" method="POST" path="/users/">
        <div className="flex flex-col gap-3">
          <div className="text-xs text-slate-500 bg-[#1a1a24] rounded-lg px-3 py-2">
            Публичный роут — токен не нужен. Пользователь создаётся через Kafka (AccountCreated event) автоматически после регистрации.
          </div>
          <Field label="Username" value={cuUser} onChange={setCuUser} placeholder="myusername" />
          <Field label="Display Name" value={cuName} onChange={setCuName} placeholder="My Name" />
          <Field label="Email (или Phone)" value={cuEmail} onChange={setCuEmail} type="email" />
          <Field label="Phone E.164 (опционально)" value={cuPhone} onChange={setCuPhone} placeholder="+79001234567" />
          <Btn loading={create.loading} onClick={() => create.call(() => usersApi.create({
            username: cuUser,
            display_name: cuName,
            email: cuEmail || undefined,
            phone: cuPhone || undefined,
          }))}>
            Создать
          </Btn>
          <ResponseBox status={create.status} data={create.data} loading={create.loading} />
        </div>
      </Card>

      {/* Get me */}
      <Card title="Мой профиль" method="GET" path="/users/me" requiresAuth>
        <div className="flex flex-col gap-3">
          <div className="text-xs text-slate-500 bg-[#1a1a24] rounded-lg px-3 py-2">
            Traefik вызывает <code className="text-violet-400">/auth/validate</code>, получает <code className="text-violet-400">X-Account-Id</code> и прокидывает в user-service автоматически.
          </div>
          <Btn loading={getMe.loading} variant="ghost" onClick={() => getMe.call(() => usersApi.getMe(at))}>
            Получить
          </Btn>
          <ResponseBox status={getMe.status} data={getMe.data} loading={getMe.loading} />
        </div>
      </Card>

      {/* Get by ID */}
      <Card title="Пользователь по ID" method="GET" path="/users/{id}">
        <div className="flex flex-col gap-3">
          <Field label="User ID (UUID)" value={targetId} onChange={setTargetId} mono placeholder="uuid" />
          <Btn loading={getById.loading} variant="ghost" onClick={() => getById.call(() => usersApi.getById(targetId))}>
            Найти
          </Btn>
          <ResponseBox status={getById.status} data={getById.data} loading={getById.loading} />
        </div>
      </Card>

      {/* Get by username */}
      <Card title="Пользователь по username" method="GET" path="/users/username/{username}">
        <div className="flex flex-col gap-3">
          <Field label="Username" value={targetUser} onChange={setTargetUser} placeholder="myusername" />
          <Btn loading={getByName.loading} variant="ghost" onClick={() => getByName.call(() => usersApi.getByUsername(targetUser))}>
            Найти
          </Btn>
          <ResponseBox status={getByName.status} data={getByName.data} loading={getByName.loading} />
        </div>
      </Card>

      {/* Batch */}
      <Card title="Пользователи по списку ID" method="POST" path="/users/batch">
        <div className="flex flex-col gap-3">
          <Field label="IDs (через запятую)" value={batchIds} onChange={setBatchIds} mono placeholder="uuid1,uuid2" />
          <Btn loading={batch.loading} variant="ghost" onClick={() => batch.call(() =>
            usersApi.getBatch(batchIds.split(',').map(s => s.trim()).filter(Boolean))
          )}>
            Получить
          </Btn>
          <ResponseBox status={batch.status} data={batch.data} loading={batch.loading} />
        </div>
      </Card>

      {/* Update profile */}
      <Card title="Обновить профиль" method="PATCH" path="/users/me/profile" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Display Name" value={profName} onChange={setProfName} placeholder="My Name" />
          <Field label="Bio (опционально)" value={profBio} onChange={setProfBio} placeholder="Немного о себе" />
          <Field label="Avatar URL (опционально)" value={profAvatar} onChange={setProfAvatar} placeholder="https://..." />
          <Field label="Version" value={profVersion} onChange={setProfVersion} type="number" />
          <Btn loading={updateProf.loading} onClick={() => updateProf.call(() => usersApi.updateProfile(at, {
            display_name: profName,
            bio: profBio || undefined,
            avatar_url: profAvatar || undefined,
            version: parseInt(profVersion),
          }))}>
            Обновить
          </Btn>
          <ResponseBox status={updateProf.status} data={updateProf.data} loading={updateProf.loading} />
        </div>
      </Card>

      {/* Update settings */}
      <Card title="Обновить настройки" method="PATCH" path="/users/me/settings" requiresAuth>
        <div className="flex flex-col gap-3">
          <SelectField label="Who Can Message" value={setMsg} onChange={setSetMsg} options={PRIVACY_OPTIONS} />
          <SelectField label="Who Can See Last Seen" value={setLastSeen} onChange={setSetLastSeen} options={PRIVACY_OPTIONS} />
          <SelectField label="Who Can See Profile" value={setProfile} onChange={setSetProfile} options={PRIVACY_OPTIONS} />
          <Field label="Language" value={setLang} onChange={setSetLang} placeholder="en" />
          <Field label="Timezone" value={setTz} onChange={setSetTz} placeholder="UTC" />
          <Field label="Version" value={setVersion} onChange={setSetVersion} type="number" />
          <Btn loading={updateSet.loading} onClick={() => updateSet.call(() => usersApi.updateSettings(at, {
            who_can_message: setMsg,
            who_can_see_last_seen: setLastSeen,
            who_can_see_profile: setProfile,
            language: setLang,
            timezone: setTz,
            version: parseInt(setVersion),
          }))}>
            Обновить
          </Btn>
          <ResponseBox status={updateSet.status} data={updateSet.data} loading={updateSet.loading} />
        </div>
      </Card>

      {/* Change email */}
      <Card title="Изменить email" method="PATCH" path="/users/me/email" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Новый Email" value={newEmail} onChange={setNewEmail} type="email" />
          <Field label="Version" value={emailVer} onChange={setEmailVer} type="number" />
          <Btn loading={chEmail.loading} onClick={() => chEmail.call(() => usersApi.changeEmail(at, {
            email: newEmail,
            version: parseInt(emailVer),
          }))}>
            Изменить
          </Btn>
          <ResponseBox status={chEmail.status} data={chEmail.data} loading={chEmail.loading} />
        </div>
      </Card>

      {/* Change phone */}
      <Card title="Изменить телефон" method="PATCH" path="/users/me/phone" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Телефон (E.164)" value={newPhone} onChange={setNewPhone} placeholder="+79001234567" />
          <Field label="Version" value={phoneVer} onChange={setPhoneVer} type="number" />
          <Btn loading={chPhone.loading} onClick={() => chPhone.call(() => usersApi.changePhone(at, {
            phone: newPhone,
            version: parseInt(phoneVer),
          }))}>
            Изменить
          </Btn>
          <ResponseBox status={chPhone.status} data={chPhone.data} loading={chPhone.loading} />
        </div>
      </Card>

      {/* Last seen */}
      <Card title="Обновить last seen" method="POST" path="/users/me/last-seen" requiresAuth>
        <div className="flex flex-col gap-3">
          <Btn loading={lastSeen.loading} variant="ghost" onClick={() => lastSeen.call(() => usersApi.updateLastSeen(at))}>
            Ping
          </Btn>
          <ResponseBox status={lastSeen.status} data={lastSeen.data} loading={lastSeen.loading} />
        </div>
      </Card>

      {/* Search */}
      <Card title="Поиск пользователей" method="GET" path="/users/search">
        <div className="flex flex-col gap-3">
          <Field label="Query" value={searchQ} onChange={setSearchQ} placeholder="alice" />
          <Field label="Limit" value={searchLimit} onChange={setSearchLimit} type="number" />
          <Btn loading={search.loading} variant="ghost" onClick={() =>
            search.call(() => usersApi.search(searchQ, parseInt(searchLimit)))
          }>
            Поиск
          </Btn>
          <ResponseBox status={search.status} data={search.data} loading={search.loading} />
        </div>
      </Card>

      {/* List */}
      <Card title="Список всех пользователей" method="GET" path="/users/me/list" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Limit" value={listLimit} onChange={setListLimit} type="number" />
          <Btn loading={list.loading} variant="ghost" onClick={() =>
            list.call(() => usersApi.list(parseInt(listLimit)))
          }>
            Получить
          </Btn>
          <ResponseBox status={list.status} data={list.data} loading={list.loading} />
        </div>
      </Card>

      {/* Delete */}
      <Card title="Удалить аккаунт" method="DELETE" path="/users/me" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Version" value={delVer} onChange={setDelVer} type="number" />
          <Btn variant="danger" loading={del.loading} onClick={() => del.call(() =>
            usersApi.deleteMe(at, parseInt(delVer))
          )}>
            Удалить аккаунт
          </Btn>
          <ResponseBox status={del.status} data={del.data} loading={del.loading} />
        </div>
      </Card>
    </div>
  )
}

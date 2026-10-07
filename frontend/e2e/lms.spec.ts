import { expect, test, type Browser, type Page } from '@playwright/test'

// One course, from the first category to the certificate, by the people who
// would do it: a SuperAdmin, an instructor, a visitor and a student. The tests
// build on each other, so they run in this order.
test.describe.configure({ mode: 'serial' })

const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? 'Admin-Passw0rd!2024'
const password = 'Passw0rd!2024'
const run = Date.now().toString().slice(-7)

const instructor = { username: `teach${run}`, email: `teach${run}@example.com` }
const student = { username: `stud${run}`, email: `stud${run}@example.com` }
const categoryName = `Category ${run}`
const courseTitle = `Go for beginners ${run}`

// a 1x1 PNG, enough for an upload
const png = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64',
)

let courseId = ''
let admin: Page
let teacher: Page
let visitor: Page
let learner: Page

async function newPage(browser: Browser): Promise<Page> {
  const context = await browser.newContext()

  return context.newPage()
}

async function logIn(page: Page, username: string, pass: string) {
  await page.goto('/login')
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password').fill(pass)
  await page.getByRole('main').getByRole('button', { name: 'Log in' }).click()
  await expect(page.getByLabel('Account menu')).toBeVisible()
}

test.beforeAll(async ({ browser }) => {
  ;[admin, teacher, visitor, learner] = await Promise.all([newPage(browser), newPage(browser), newPage(browser), newPage(browser)])
})

test.afterAll(async () => {
  await Promise.all([admin, teacher, visitor, learner].map((page) => page.context().close()))
})

test.describe('SuperAdmin prepares the platform', () => {
  test('logs in and lands on the dashboard', async () => {
    await logIn(admin, 'admin', adminPassword)
    await expect(admin).toHaveURL(/\/dashboard$/)
    await expect(admin.getByText('Revenue')).toBeVisible()
  })

  test('creates a category', async () => {
    await admin.goto('/admin/categories')
    await admin.getByRole('button', { name: 'New category' }).click()
    await admin.getByLabel('Name').fill(categoryName)
    await admin.getByRole('button', { name: 'Save' }).click()
    await expect(admin.getByRole('cell', { name: categoryName })).toBeVisible()
  })

  test('creates an instructor and finds them with the search', async () => {
    await admin.goto('/admin/users')
    await admin.getByRole('button', { name: 'New user' }).click()

    const dialog = admin.getByRole('dialog')
    await dialog.getByLabel('First name').fill('Tina')
    await dialog.getByLabel('Last name').fill('Teacher')
    await dialog.getByLabel('Username').fill(instructor.username)
    await dialog.getByLabel('Email').fill(instructor.email)
    await dialog.getByLabel('Password').fill(password)
    await dialog.getByLabel('Role').selectOption('Instructor')
    await dialog.getByRole('button', { name: 'Save' }).click()
    await expect(dialog).toBeHidden()

    await admin.getByLabel('Search users').fill(instructor.username)
    await expect(admin.getByRole('row')).toHaveCount(2) // the header and the user
    await expect(admin.getByText(instructor.email)).toBeVisible()
  })
})

test.describe('the instructor builds the course', () => {
  test('logs in; the admin pages are closed to them', async () => {
    await logIn(teacher, instructor.username, password)
    await teacher.goto('/admin/users')
    await expect(teacher).not.toHaveURL(/\/admin/)
  })

  test('creates a course with a cover', async () => {
    await teacher.goto('/teach/courses/new')

    await teacher.getByLabel('Title').fill(courseTitle)
    await teacher.getByLabel('Description').fill('Learn Go from zero.')
    await teacher.getByLabel('Category').selectOption({ label: categoryName })
    await teacher.getByLabel('Level').selectOption('beginner')
    await teacher.getByLabel('Price (USD, 0 is free)').fill('25')
    await teacher.getByLabel('Duration (minutes)').fill('90')
    await teacher.getByLabel('What students learn').fill('Write Go programs\nUse goroutines')
    await teacher.getByLabel('Requirements').fill('A laptop')
    await teacher.locator('#upload-course_cover').setInputFiles({ name: 'cover.png', mimeType: 'image/png', buffer: png })
    await expect(teacher.locator('img[src^="blob:"]').first()).toBeVisible()

    await teacher.getByRole('button', { name: 'Create the course' }).click()
    await expect(teacher).toHaveURL(/\/teach\/courses\/[0-9a-f-]{36}$/)

    courseId = teacher.url().split('/').pop()!
    await expect(teacher.getByRole('heading', { name: courseTitle })).toBeVisible()
    await expect(teacher.locator('img[src*="covers/"]').first()).toBeVisible()
  })

  test('adds modules and lessons: one free preview, one locked', async () => {
    await teacher.goto(`/teach/courses/${courseId}?tab=content`)

    await teacher.getByRole('button', { name: 'Add a module' }).click()
    await teacher.getByLabel('Title').fill('Basics')
    await teacher.getByRole('button', { name: 'Save' }).click()
    await expect(teacher.getByText('1. Basics')).toBeVisible()

    await teacher.getByRole('button', { name: 'Add a lesson' }).click()
    await teacher.getByLabel('Title').fill('Hello world')
    await teacher.getByLabel('Duration (minutes)').fill('10')
    await teacher.getByLabel('Free preview: visitors can open this lesson').check()
    await teacher.getByRole('button', { name: 'Save' }).click()
    await expect(teacher.getByText('1. Hello world')).toBeVisible()

    await teacher.getByRole('button', { name: 'Add a module' }).click()
    await teacher.getByLabel('Title').fill('Advanced')
    await teacher.getByRole('button', { name: 'Save' }).click()
    await expect(teacher.getByText('2. Advanced')).toBeVisible()

    await teacher.getByRole('button', { name: 'Add a lesson' }).last().click()
    await teacher.getByLabel('Title').fill('Deep dive')
    await teacher.getByRole('button', { name: 'Save' }).click()
    await expect(teacher.getByText('1. Deep dive')).toBeVisible()
  })

  test('adds a text material to each lesson', async () => {
    await teacher.getByRole('button', { name: 'Materials' }).first().click()
    await teacher.getByLabel('Text', { exact: true }).fill('fmt.Println("hello")')
    await teacher.getByRole('button', { name: 'Add the material' }).click()
    await expect(teacher.getByText('fmt.Println("hello")')).toBeVisible()
    await teacher.keyboard.press('Escape')

    await teacher.getByRole('button', { name: 'Materials' }).last().click()
    await teacher.getByLabel('Text', { exact: true }).fill('the secret part')
    await teacher.getByRole('button', { name: 'Add the material' }).click()
    await expect(teacher.getByText('the secret part')).toBeVisible()
    await teacher.keyboard.press('Escape')
  })

  test('adds a final quiz with one question', async () => {
    await teacher.goto(`/teach/courses/${courseId}?tab=quizzes`)

    await teacher.getByRole('button', { name: 'New quiz' }).click()
    await teacher.getByLabel('Title').fill('Final test')
    await teacher.getByLabel('Pass at %').fill('50')
    await teacher.getByRole('button', { name: 'Save' }).click()
    await expect(teacher.getByText('Final test')).toBeVisible()

    await teacher.getByRole('button', { name: 'Questions' }).click()
    await teacher.getByLabel('Question', { exact: true }).fill('Is Go compiled?')
    await teacher.getByLabel('Option 1', { exact: true }).fill('Yes')
    await teacher.getByLabel('Option 2', { exact: true }).fill('No')
    await teacher.getByRole('button', { name: 'Add the question' }).click()
    await expect(teacher.getByText('1. Is Go compiled?')).toBeVisible()
    await teacher.keyboard.press('Escape')
  })

  test('a draft is not in the catalog or on the home page, not even for the SuperAdmin', async () => {
    await admin.goto(`/courses?q=${encodeURIComponent(courseTitle)}`)
    await expect(admin.getByText('No course matches your search')).toBeVisible()

    await admin.goto('/')
    await expect(admin.getByText(courseTitle)).toHaveCount(0)
    await admin.goto('/dashboard')
    await expect(admin.getByText(courseTitle)).toHaveCount(0)
  })

  test('publishes the course', async () => {
    await teacher.getByRole('button', { name: 'Publish', exact: true }).click()
    await expect(teacher.getByText('Published', { exact: true })).toBeVisible()
  })
})

test.describe('a visitor', () => {
  test('finds the course by its title and opens it', async () => {
    await visitor.goto(`/courses?q=${encodeURIComponent(courseTitle)}`)
    await expect(visitor.getByText('1 course', { exact: true })).toBeVisible()

    await visitor.getByRole('link', { name: new RegExp(courseTitle) }).click()
    await expect(visitor.getByText('Write Go programs')).toBeVisible()
    await expect(visitor.getByRole('link', { name: 'Log in to enroll' })).toBeVisible()

    // the names of all lessons are shown; only the free preview can be opened
    await expect(visitor.getByText('Deep dive')).toBeVisible()
    await expect(visitor.getByLabel('Locked')).toHaveCount(1)
  })

  test('opens the free preview lesson, but not the locked one', async () => {
    await visitor.getByRole('link', { name: /Hello world/ }).click()
    await expect(visitor).toHaveURL(/\/preview\//)
    await expect(visitor.getByText('fmt.Println("hello")')).toBeVisible()
    await expect(visitor.getByText('Like what you see?')).toBeVisible()
    await expect(visitor.getByLabel('Locked')).toBeVisible()
    await expect(visitor.getByText('the secret part')).toHaveCount(0)
  })

  test('the home page shows the categories and the popular courses', async () => {
    await visitor.goto('/')
    await expect(visitor.getByRole('heading', { name: /Learn something new/ })).toBeVisible()
    await expect(visitor.getByRole('region', { name: 'Categories' })).toBeVisible()
    await expect(visitor.getByRole('region', { name: 'Bestsellers' }).getByRole('link').first()).toBeVisible()

    // the search of the home page leads to the catalog
    await visitor.getByLabel('Search courses').fill(courseTitle)
    await visitor.getByRole('button', { name: 'Search' }).click()
    await expect(visitor).toHaveURL(/\/courses\?q=/)
    await expect(visitor.getByText('1 course', { exact: true })).toBeVisible()
  })

  test('sees nothing when the search finds nothing', async () => {
    await visitor.goto('/courses?q=zzzz-no-such-course')
    await expect(visitor.getByText('No course matches your search')).toBeVisible()
  })
})

test.describe('a student', () => {
  test('signs up and sees the form errors first', async () => {
    await learner.goto('/register')
    await learner.getByRole('main').getByRole('button', { name: 'Sign up' }).click()
    await expect(learner.getByText('First name is required')).toBeVisible()

    await learner.getByLabel('First name').fill('Sam')
    await learner.getByLabel('Last name').fill('Student')
    await learner.getByLabel('Username').fill(student.username)
    await learner.getByLabel('Email').fill(student.email)
    await learner.getByLabel('Password').fill(password)
    await learner.getByRole('main').getByRole('button', { name: 'Sign up' }).click()

    await expect(learner).toHaveURL(/\/dashboard$/)
    await expect(learner.getByText('Hello, Sam Student')).toBeVisible()
  })

  test('enrolls and studies the lesson', async () => {
    await learner.goto(`/courses/${courseId}`)
    await learner.getByRole('button', { name: 'Enroll for $25.00' }).click()

    await expect(learner).toHaveURL(/\/lessons\//)
    await expect(learner.getByText('fmt.Println("hello")')).toBeVisible()

    // marking a lesson done opens the next one
    await learner.getByRole('button', { name: 'Mark as done' }).click()
    await expect(learner.getByText('the secret part')).toBeVisible()
    await learner.getByRole('button', { name: 'Mark as done' }).click()
    await expect(learner.getByRole('button', { name: 'Mark as not done' })).toBeVisible()
    await learner.goto('/my-courses')
    await expect(learner.getByText('100%')).toBeVisible()
  })

  test('takes the final quiz and passes it', async () => {
    await learner.goto(`/learn/${courseId}`)
    await learner.getByRole('link', { name: 'Final test' }).click()
    await learner.getByRole('button', { name: 'Start the quiz' }).click()

    await expect(learner.getByText('1. Is Go compiled?')).toBeVisible()
    // the options come in a random order, so the answer is found by its text
    await learner.getByLabel('Yes').check()
    await learner.getByRole('button', { name: 'Finish the quiz' }).click()

    await expect(learner.getByText('You passed')).toBeVisible()
  })

  test('gets the certificate, downloads it and it can be checked by anybody', async () => {
    await learner.goto('/certificates')
    await expect(learner.getByText(courseTitle).first()).toBeVisible()

    const download = learner.waitForEvent('download')
    await learner.getByRole('button', { name: 'Download PDF' }).click()
    expect((await download).suggestedFilename()).toMatch(/\.pdf$/)

    const number = (await learner.locator('p.font-mono').first().textContent())!.trim()

    await visitor.goto(`/verify/${number}`)
    await expect(visitor.getByText('This certificate is real')).toBeVisible()
    await expect(visitor.getByText('Sam Student')).toBeVisible()
  })

  test('reviews the course', async () => {
    await learner.goto(`/courses/${courseId}`)
    await learner.getByLabel('Your review (optional)').fill('Great course')
    await learner.getByRole('button', { name: 'Send the review' }).click()
    await expect(learner.getByText('Great course')).toBeVisible()
  })

  test('uploads a photo', async () => {
    await learner.goto('/profile')
    await learner.locator('#upload-avatar').setInputFiles({ name: 'me.png', mimeType: 'image/png', buffer: png })
    await expect(learner.getByText('Photo saved')).toBeVisible()
    await expect(learner.getByLabel('Account menu').locator('img')).toBeVisible()
  })

  test('changes the profile', async () => {
    await learner.goto('/profile')
    await learner.getByLabel('About you').fill('I like Go')
    await learner.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(learner.getByText('Profile saved')).toBeVisible()
  })
})

test.describe('afterwards', () => {
  test('the instructor sees the student with the progress', async () => {
    await teacher.goto(`/teach/courses/${courseId}?tab=students`)
    await expect(teacher.getByText('Sam Student')).toBeVisible()
    await expect(teacher.getByText('2 of 2 lessons')).toBeVisible()
  })

  test('the SuperAdmin sees the money and the reports', async () => {
    await admin.goto('/admin/payments')
    await expect(admin.getByRole('cell', { name: '$25.00' }).first()).toBeVisible()

    // the period is picked from a calendar
    await admin.locator('#range-from').click()
    await admin.getByRole('button', { name: 'Today', exact: true }).click()
    await expect(admin.locator('#range-from')).not.toContainText('Any day')
    await expect(admin.getByRole('cell', { name: '$25.00' }).first()).toBeVisible()

    // and the dashboard has the same filter for the latest payments
    await admin.goto('/dashboard')
    await admin.getByRole('button', { name: 'Last 30 days' }).click()
    await expect(admin.getByRole('region', { name: 'Latest payments' }).getByRole('cell', { name: '$25.00' }).first()).toBeVisible()

    await admin.goto('/admin/finance')
    await expect(admin.getByText('Net profit').first()).toBeVisible()

    await admin.goto('/admin/reports')
    await admin.getByLabel('Report').selectOption('revenue')
    await expect(admin.getByRole('cell', { name: courseTitle })).toBeVisible()
  })

  test('blocking the student signs them out at once', async () => {
    await admin.goto('/admin/users')
    await admin.getByLabel('Search users').fill(student.username)
    await expect(admin.getByRole('row')).toHaveCount(2)
    await admin.getByRole('button', { name: /More actions for/ }).click()
    await admin.getByRole('menuitem', { name: 'Block' }).click()
    await expect(admin.getByRole('cell', { name: 'Blocked' })).toBeVisible()

    // an access token carries its time in seconds; the block counts from the next one
    await learner.waitForTimeout(1200)
    await learner.goto('/my-courses')
    await expect(learner).toHaveURL(/\/login/)
  })
})

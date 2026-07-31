import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/packages' },
  { path: '/packages', name: 'packages', component: () => import('../views/PackageView.vue') },
  { path: '/deploy/:id', name: 'deploy', component: () => import('../views/DeployView.vue') },
  { path: '/history', name: 'history', component: () => import('../views/HistoryView.vue') }
]

export default createRouter({
  history: createWebHistory(),
  routes
})

document.getElementById('js').textContent = 'js ok';
window.addEventListener('click', (e) => {
  if (e.target.id === 'cancelled') e.preventDefault();
});

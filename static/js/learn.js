(() => {
  const list = document.getElementById('article-list');
  const article = document.getElementById('article-title');
  const getJSON = async path => {
    const response = await fetch(path, {credentials:'same-origin'});
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Could not load this page');
    return data;
  };
  const addText = (parent, tag, value, className) => {
    const element = document.createElement(tag);
    element.textContent = value;
    if (className) element.className = className;
    parent.append(element);
    return element;
  };
  if (list) {
    getJSON('/api/v1/articles').then(({articles}) => {
      document.getElementById('articles-status').textContent = articles.length ? '' : 'New guides are on the way.';
      for (const item of articles) {
        const card = document.createElement('a'); card.className = 'article-tile'; card.href = '/learn/' + encodeURIComponent(item.slug);
        addText(card,'span',item.category,'tile-category');
        addText(card,'h2',item.title);
        addText(card,'p',item.summary);
        addText(card,'strong','Read guide →');
        list.append(card);
      }
    }).catch(error => { document.getElementById('articles-status').textContent = error.message; });
  }
  if (!article) return;
  const slug = decodeURIComponent(location.pathname.split('/').pop());
  const comments = document.getElementById('comments');
  const status = document.getElementById('comments-status');
  const renderComments = async () => {
    const data = await getJSON('/api/v1/articles/' + encodeURIComponent(slug) + '/comments');
    comments.replaceChildren();
    status.textContent = data.comments.length ? '' : 'No comments yet. Start the conversation.';
    for (const item of data.comments) {
      const card = document.createElement('article'); card.className = 'comment-card';
      addText(card,'strong',item.name);
      addText(card,'time','#' + item.id + ' · ' + new Date(item.created_at).toLocaleDateString());
      addText(card,'p',item.body);
      comments.append(card);
    }
  };
  getJSON('/api/v1/articles/' + encodeURIComponent(slug)).then(data => {
    document.title = data.title + ' · L2EStudyLink';
    document.getElementById('article-status').textContent = '';
    document.getElementById('article-category').textContent = data.category;
    article.textContent = data.title;
    document.getElementById('article-summary').textContent = data.summary;
    const body = document.getElementById('article-body');
    for (const paragraph of data.body.split(/\n\s*\n/)) addText(body,'p',paragraph);
    return renderComments();
  }).catch(error => { document.getElementById('article-status').textContent = error.message; status.textContent = ''; });
  // A public reader stays on this page: only a successful /me opens the form.
  fetch('/api/v1/me',{credentials:'same-origin'}).then(response => {
    document.getElementById(response.ok ? 'comment-form' : 'join-prompt').hidden = false;
  }).catch(() => { document.getElementById('join-prompt').hidden = false; });
  document.getElementById('comment-form').addEventListener('submit',async event => {
    event.preventDefault();
    const form=event.currentTarget, button=form.querySelector('button'), message=document.getElementById('post-status');
    button.disabled=true; message.textContent='Posting…';
    try {
      const response=await fetch('/api/v1/articles/' + encodeURIComponent(slug) + '/comments',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({body:document.getElementById('comment-body').value})});
      const data=await response.json();
      if (response.status===401) {document.getElementById('join-prompt').hidden=false;form.hidden=true;return;}
      if (!response.ok) throw new Error(data.error || 'Could not post comment');
      form.reset(); message.textContent='Your comment is live.'; await renderComments();
    } catch(error) { message.textContent=error.message; } finally {button.disabled=false;}
  });
})();

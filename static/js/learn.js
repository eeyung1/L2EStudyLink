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
      const readingList=articles.length>1 ? articles.filter(item=>item.slug!=='start-an-agentic-workflow') : articles;
      document.getElementById('articles-status').textContent = readingList.length ? '' : 'New guides are on the way.';
      for (const item of readingList) {
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
  const articleReturn = encodeURIComponent(location.pathname);
  for (const link of document.querySelectorAll('.article-signup, #join-prompt a[href="/page/signup"]')) link.href='/page/signup?next='+articleReturn;
  const signInLink = document.querySelector('#join-prompt a[href="/page/login"]');
  if (signInLink) signInLink.href='/page/login?next='+articleReturn;
  const comments = document.getElementById('comments');
  const status = document.getElementById('comments-status');
  let canReply = false;
  const postComment = async (body,parentID) => {
    const response=await fetch('/api/v1/articles/' + encodeURIComponent(slug) + '/comments',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({body,parent_id:parentID})});
    const data=await response.json();
    if (response.status===401) {
      document.getElementById('join-prompt').hidden=false;
      document.getElementById('comment-form').hidden=true;
      canReply=false;
    }
    if (!response.ok) throw new Error(data.error || 'Could not post comment');
    return data;
  };
  const commentCard = (item, rootID) => {
    const card=document.createElement('article');card.className='comment-card';
    addText(card,'strong',item.name);
    addText(card,'time','#' + item.id + ' · ' + new Date(item.created_at).toLocaleDateString());
    addText(card,'p',item.body);
    const reply=document.createElement('button');reply.type='button';reply.className='reply-action';reply.dataset.replyTo=rootID;reply.dataset.replyName=item.name;reply.textContent='↳ Reply';card.append(reply);
    return card;
  };
  const renderComments = async () => {
    const data = await getJSON('/api/v1/articles/' + encodeURIComponent(slug) + '/comments');
    comments.replaceChildren();
    status.textContent = data.comments.length ? '' : 'No comments yet. Start the conversation.';
    const byID=new Map(data.comments.map(item=>[item.id,item]));
    const roots=[],replies=new Map();
    for (const item of data.comments) {
      if (item.parent_id && byID.has(item.parent_id)) {
        if (!replies.has(item.parent_id)) replies.set(item.parent_id,[]);
        replies.get(item.parent_id).push(item);
      } else roots.push(item);
    }
    for (const item of roots) {
      const thread=document.createElement('section');thread.className='thread';thread.id='comment-'+item.id;
      thread.append(commentCard(item,item.id));
      const children=replies.get(item.id)||[];
      if (children.length) {
        const nested=document.createElement('div');nested.className='thread-replies';
        for (const child of children.reverse()) nested.append(commentCard(child,item.id));
        thread.append(nested);
      }
      comments.append(thread);
    }
  };
  comments.addEventListener('click',event => {
    const button=event.target.closest('.reply-action');
    if (!button) return;
    if (!canReply) {
      document.getElementById('join-prompt').scrollIntoView({behavior:'smooth'});
      return;
    }
    document.querySelectorAll('.reply-form').forEach(form=>form.remove());
    const rootID=Number(button.dataset.replyTo);
    const form=document.createElement('form');form.className='reply-form';
    const label=addText(form,'label','Reply to '+button.dataset.replyName);
    const input=document.createElement('textarea');input.required=true;input.minLength=3;input.maxLength=1200;input.placeholder='Share an answer or ask a follow-up question';input.id='reply-'+rootID;label.htmlFor=input.id;form.append(input);
    const actions=document.createElement('div');actions.className='comment-actions';
    const cancel=addText(actions,'button','Cancel','cancel-reply');cancel.type='button';cancel.addEventListener('click',()=>form.remove());
    const submit=addText(actions,'button','Post reply','button button-gold');submit.type='submit';actions.append(submit);form.append(actions);
    const message=addText(form,'p','');message.setAttribute('role','status');
    form.addEventListener('submit',async e=>{
      e.preventDefault();submit.disabled=true;message.textContent='Posting…';
      try {await postComment(input.value,rootID);await renderComments();document.getElementById('comment-'+rootID)?.scrollIntoView({behavior:'smooth',block:'center'});}
      catch(error){message.textContent=error.message;submit.disabled=false;}
    });
    button.closest('.thread').append(form);input.focus();
  });
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
    canReply=response.ok;
    document.getElementById(response.ok ? 'comment-form' : 'join-prompt').hidden = false;
  }).catch(() => { document.getElementById('join-prompt').hidden = false; });
  document.getElementById('comment-form').addEventListener('submit',async event => {
    event.preventDefault();
    const form=event.currentTarget, button=form.querySelector('button'), message=document.getElementById('post-status');
    button.disabled=true; message.textContent='Posting…';
    try {
      await postComment(document.getElementById('comment-body').value,null);
      form.reset(); message.textContent='Your comment is live.'; await renderComments();
    } catch(error) { message.textContent=error.message; } finally {button.disabled=false;}
  });
})();

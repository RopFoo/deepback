import React, { useState, useEffect } from 'react';
import Letter from '../modules/Letter';
import axios from 'axios';
import { useLocation } from 'react-router-dom';

const Answer: React.FC = () => {
  const location = useLocation();
  const [question, setQuestion] = useState({ title: '', body: '' });

  useEffect(() => {
    const url = `http://localhost:8000/api${location.pathname}`;
    axios
      .get(url)
      .then((response) => {
        setQuestion(response.data);
      })
      .catch((err) => console.log(err));
  }, []);

  return (
    <div>
      <div className='c-question-scope'>
        <h1>{question.title}</h1>
        <p>{question.body}</p>
        <a href='#answer' className='c-btn__toAnswer'>
          answer
        </a>
      </div>
      <div id='answer' className='c-answer-scope'>
        <h1>{question.title}</h1>
        <Letter />
      </div>
    </div>
  );
};

export default Answer;

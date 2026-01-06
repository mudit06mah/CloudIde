import React from 'react'

export default function Home(){
    const handleFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
    }
    return (
        <>
            <div className="p-4 justify-center items-center border-2 border-gray-300 rounded-lg shadow-md max-w-md mx-auto mt-10">
                <form className="space-y-2" onSubmit={handleFormSubmit}>
                    <input type="radio" name="option" value="1" />
                    <label>C++</label><br />
                    <input type="radio" name="option" value="2" />
                    <label>Python</label><br />
                    <input type="radio" name="option" value="3" />
                    <label>JavaScript</label><br />
                    <input type="radio" name="option" value="4" />
                    <label>Golang</label><br />
                    <input type="radio" name="option" value="5" />
                    <label>Nodejs</label><br />
                    <input type="radio" name="option" value="6" />
                    <label>React</label><br />
                    <button type="submit" className="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
                </form>
            </div>
        </>
    )
}